"""Structured JSON logs with the same schema as the server (timestamp, level, service, event, context).

``LOG_LEVEL`` is ERROR, WARN, INFO (default) or DEBUG. Keys that could carry a secret or private content
are masked at every level, and exception text is never written: only a code or the exception class.
Free text values are scrubbed too (tokens, Bearer headers, URLs with credentials or signatures, clip paths),
so the same lines can be shipped to the server as the job's logs.
"""

from __future__ import annotations

import json
import logging
import os
import re
import secrets
import sys
import threading
from collections import deque
from datetime import datetime, timezone
from typing import Any, Callable, TextIO

SERVICE = "worker"
LOGGER_NAME = "snitch_worker"
MASKED = "[masked]"
FORBIDDEN = ("authorization", "token", "cookie", "password", "secret", "signed_url", "transcript", "embedding", "fingerprint", "api_key", "apikey", "bearer", "credential")
# Keys that carry free error text: the value is never written, only a code and a category (see failure()).
RAW_ERROR_KEYS = ("error", "err", "reason", "exception", "detail")
# Lines of one job reach the server by batches: at most SHIP_BATCH lines, at least every SHIP_INTERVAL seconds.
SHIP_BATCH = 50
SHIP_INTERVAL = 2.0
# The Worker's own journal (ADR 0063): the server keeps a bounded window, the Worker a bounded buffer for the
# time Snitch is out of reach. The server refuses a line over 4096 bytes and a batch over 200 lines.
UPLINK_BATCH = 200
UPLINK_BATCH_BYTES = 256 * 1024  # a batch is cut by size too: the server limits the body, and JSON escaping can double it
UPLINK_BUFFER = 2000
UPLINK_LINE_BYTES = 4000
UPLINK_BACKOFF_CAP = 30.0
_LEVELS = {"ERROR": logging.ERROR, "WARN": logging.WARNING, "WARNING": logging.WARNING, "INFO": logging.INFO, "DEBUG": logging.DEBUG}
_URL = re.compile(r"(?i)\b([a-z][a-z0-9+.-]*://)[^\s\"]*")
_SECRETS = (
    # A Hugging Face token is "hf_" and letters or digits; "hf_not_found" is a stable code, not a token.
    (re.compile(r"\bhf_[A-Za-z0-9]{6,}"), MASKED),
    (re.compile(r"(?i)\bbearer\s+[^\s\"',;]+"), "Bearer " + MASKED),
    (re.compile(r"\b(?:lt|rw)_[A-Za-z0-9][A-Za-z0-9_-]{7,}"), MASKED),
)
_PATHS = (
    (re.compile(r"(?:[A-Za-z]:)?/(?:tmp|private|var|Users|home|opt|srv|mnt|data|Volumes)/[^\s\"]*"), "[masked_path]"),
    (re.compile(r"[^\s\"/]*\.(?:wav|mp3|m4a|mp4|mov|mkv|webm|ogg|flac|aac|media)\b", re.IGNORECASE), "[masked_path]"),
)


def parse_level(value: str | None) -> int:
    return _LEVELS.get((value or "").strip().upper(), logging.INFO)


def _sensitive(key: str) -> bool:
    lowered = key.lower()
    return any(fragment in lowered for fragment in FORBIDDEN)


def failure(error: BaseException) -> dict[str, str]:
    """Stable ``error_code`` and ``error_category`` for an exception, never its text."""
    if isinstance(error, TimeoutError):
        return {"error_code": "timeout", "error_category": "timeout"}
    if isinstance(error, OSError):
        return {"error_code": "io_error", "error_category": "io"}
    return {"error_code": "operation_error", "error_category": "internal"}


def _url(match: re.Match[str]) -> str:
    # The scheme and the host only: no userinfo, path or query survives.
    rest = match.group(0)[len(match.group(1)):]
    host = re.split(r"[/?#]", rest, maxsplit=1)[0].rsplit("@", 1)[-1]
    return match.group(1) + host + ("/[masked]" if re.search(r"[/?#]", rest) else "")


def scrub(text: str) -> str:
    """Mask credentials, credentialed or signed URLs and clip paths in a free text value."""
    text = _URL.sub(_url, text)
    for pattern, replacement in _SECRETS + _PATHS:
        text = pattern.sub(replacement, text)
    return text


def _clean(key: str, value: Any) -> Any:
    if _sensitive(key):
        return MASKED
    if key.lower() in RAW_ERROR_KEYS and value is not None:
        if isinstance(value, str):
            return MASKED
        # An exception or a structure may carry free text: only a code and a category survive.
        described = failure(value) if isinstance(value, BaseException) else failure(Exception())
        return {"code": described["error_code"], "category": described["error_category"]}
    if isinstance(value, BaseException):
        return type(value).__name__
    if isinstance(value, dict):
        return {str(k): _clean(str(k), v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [_clean(key, item) for item in value]
    if isinstance(value, str):
        return scrub(value)
    if isinstance(value, (int, float, bool)) or value is None:
        return value
    return scrub(str(value))


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": datetime.fromtimestamp(record.created, timezone.utc).isoformat(timespec="milliseconds"),
            "level": "WARN" if record.levelno == logging.WARNING else record.levelname,
            "service": SERVICE,
            "event": scrub(record.getMessage()),
        }
        for key, value in getattr(record, "fields", {}).items():
            if key not in payload:
                payload[key] = _clean(key, value)
        # Exception text can echo URLs, paths or tokens: keep the class only.
        if record.exc_info and record.exc_info[0] is not None:
            payload["error_class"] = record.exc_info[0].__name__
        return json.dumps(payload, ensure_ascii=False)


def configure(level: str | None = None, stream: TextIO | None = None) -> logging.Logger:
    """Install the JSON handler on the Worker's logger; ``level`` defaults to the LOG_LEVEL variable."""
    logger = logging.getLogger(LOGGER_NAME)
    for handler in list(logger.handlers):
        logger.removeHandler(handler)
    handler = logging.StreamHandler(stream or sys.stderr)
    handler.setFormatter(JsonFormatter())
    logger.addHandler(handler)
    logger.setLevel(parse_level(os.environ.get("LOG_LEVEL") if level is None else level))
    logger.propagate = False
    return logger


def get_logger() -> logging.Logger:
    return logging.getLogger(LOGGER_NAME)


def emit(level: int, event: str, **fields: Any) -> None:
    get_logger().log(level, event, extra={"fields": fields})


class LineShipper(logging.Handler):
    """Keeps the JSON lines of one job and hands them to ``send`` in batches of at most SHIP_BATCH lines,
    at least every ``interval`` seconds. A failed send drops its batch: shipping never affects the job."""

    def __init__(self, job_id: int, send: Callable[[list[str]], None], interval: float = SHIP_INTERVAL) -> None:
        super().__init__(level=logging.NOTSET)
        self.setFormatter(JsonFormatter())
        self._job_id = job_id
        self._send = send
        self._interval = interval
        self._lines: list[str] = []
        self._lock = threading.Lock()
        self._full = threading.Event()
        self._stopped = threading.Event()
        self._thread = threading.Thread(target=self._run, name="job-log-shipper", daemon=True)
        self._thread.start()

    def emit(self, record: logging.LogRecord) -> None:
        if getattr(record, "fields", {}).get("job_id") != self._job_id:
            return
        try:
            line = self.format(record)
        except Exception:  # noqa: BLE001 - a log line must never break the job
            return
        with self._lock:
            self._lines.append(line)
            full = len(self._lines) >= SHIP_BATCH
        if full:
            self._full.set()

    def _run(self) -> None:
        while not self._stopped.is_set():
            self._full.wait(self._interval)
            self._full.clear()
            self._drain()

    def flush(self) -> None:
        """Send the lines kept so far, now."""
        self._drain()

    def _drain(self) -> None:
        while True:
            with self._lock:
                batch, self._lines = self._lines[:SHIP_BATCH], self._lines[SHIP_BATCH:]
            if not batch:
                return
            try:
                self._send(batch)
            except Exception:  # noqa: BLE001 - dropped on purpose: the Worker keeps running
                pass

    def close(self) -> None:
        if not self._stopped.is_set():
            self._stopped.set()
            self._full.set()
            self._thread.join(timeout=5)
            self._drain()
        super().close()


class BatchRefused(Exception):
    """Snitch refused a batch for good (not a network or availability problem): resending it would never work."""


class WorkerLogUplink(logging.Handler):
    """Sends every line of the Worker's journal to Snitch, from its start, so an administrator reads it remotely.

    The lines are the scrubbed JSON lines the Worker already writes. They are numbered per process (the session),
    so a batch sent twice because an answer was lost is stored once. Snitch out of reach never blocks the Worker:
    lines wait in a bounded buffer, the oldest ones go first, and a failure is never logged (it would feed itself).
    The handler may be attached before the client exists: lines emitted meanwhile wait for ``start``."""

    def __init__(self, session_id: str | None = None, interval: float = SHIP_INTERVAL) -> None:
        super().__init__(level=logging.NOTSET)
        self.setFormatter(JsonFormatter())
        self.session_id = session_id or secrets.token_urlsafe(12)
        self._interval = interval
        self._pending: deque[tuple[int, str]] = deque()
        self._next_seq = 0
        self._lock = threading.Lock()
        self._wake = threading.Event()
        self._stopped = threading.Event()
        self._send: Callable[[str, list[tuple[int, str]]], None] | None = None
        self._thread: threading.Thread | None = None

    def emit(self, record: logging.LogRecord) -> None:
        try:
            line = self.format(record)
        except Exception:  # noqa: BLE001 - a log line must never break the Worker
            return
        # The limit is in bytes (the server counts bytes); a cut inside a character is dropped, not kept broken.
        raw = line.encode("utf-8")
        if len(raw) > UPLINK_LINE_BYTES:
            line = raw[:UPLINK_LINE_BYTES].decode("utf-8", "ignore")
        with self._lock:
            self._pending.append((self._next_seq, line))
            self._next_seq += 1
            while len(self._pending) > UPLINK_BUFFER:
                self._pending.popleft()
            full = len(self._pending) >= UPLINK_BATCH
        if full:
            self._wake.set()

    def start(self, send: Callable[[str, list[tuple[int, str]]], None]) -> None:
        """Begin sending. ``send(session_id, [(seq, line), ...])`` raises when Snitch did not take the batch,
        and ``BatchRefused`` when it never will (that batch is dropped, so it cannot block the ones behind it)."""
        if self._thread is not None:
            return
        self._send = send
        self._thread = threading.Thread(target=self._run, name="worker-log-uplink", daemon=True)
        self._thread.start()

    def _run(self) -> None:
        delay = self._interval
        while not self._stopped.is_set():
            self._wake.wait(delay)
            self._wake.clear()
            delay = self._interval if self._drain() else min(max(delay * 2, self._interval), UPLINK_BACKOFF_CAP)

    def _drain(self) -> bool:
        """Send what is waiting, batch by batch. False when Snitch refused a batch: it stays and is tried again."""
        send = self._send
        while send is not None:
            with self._lock:
                batch, size = [], 0
                for item in self._pending:
                    size += len(item[1]) + 32
                    if batch and (len(batch) >= UPLINK_BATCH or size > UPLINK_BATCH_BYTES):
                        break
                    batch.append(item)
            if not batch:
                return True
            try:
                send(self.session_id, batch)
            except BatchRefused:
                pass  # dropped on purpose, below: a batch Snitch will never take must not block the others
            except Exception:  # noqa: BLE001 - kept for the next try; the Worker keeps running
                return False
            with self._lock:
                # Only what was sent goes: lines dropped meanwhile by the buffer limit are already gone.
                sent = {seq for seq, _ in batch}
                while self._pending and self._pending[0][0] in sent:
                    self._pending.popleft()
        return True

    def close(self) -> None:
        if not self._stopped.is_set():
            self._stopped.set()
            self._wake.set()
            if self._thread is not None:
                self._thread.join(timeout=5)
            self._drain()  # one last try, so the end of the session (shutdown, restart) reaches Snitch
        super().close()
