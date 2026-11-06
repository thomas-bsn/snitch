"""Worker-side validation and active transcription configuration."""

from __future__ import annotations

import re
import threading
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from urllib.parse import urlparse, urlsplit, urlunsplit

import httpx
from snitch_client.client import AuthenticatedClient


@dataclass(frozen=True)
class ActiveConfiguration:
    version: str
    endpoint: str
    model: str
    language: str


_lock = threading.Lock()
_active: ActiveConfiguration | None = None


def active_configuration() -> ActiveConfiguration | None:
    with _lock:
        return _active


def activate(configuration: ActiveConfiguration) -> None:
    global _active
    with _lock:
        _active = configuration


def test_configuration(endpoint: str, model: str) -> str | None:
    """Return a stable error code, or None when the model is advertised."""
    parsed = urlparse(endpoint)
    if parsed.scheme not in ("http", "https") or not parsed.netloc or not model:
        return "connection_failed"
    try:
        response = httpx.get(endpoint.rstrip("/") + "/v1/models", timeout=5.0)
        response.raise_for_status()
        payload = response.json()
    except (httpx.HTTPError, ValueError):
        return "connection_failed"
    models = payload.get("data") if isinstance(payload, dict) else None
    if not isinstance(models, list):
        return "connection_failed"
    return None if any(isinstance(item, dict) and item.get("id") == model for item in models) else "model_missing"


def sync_configuration(client: AuthenticatedClient) -> None:
    """Receive and validate the latest server-owned configuration before claiming."""
    response = client.get_httpx_client().get("/api/worker/transcription-config")
    if response.status_code == 404:
        return  # A worker binary may briefly talk to a server predating this contract.
    response.raise_for_status()
    desired = response.json()
    if desired.get("mode") != "enabled":
        reachability.clear_target()
        return
    version = desired.get("version")
    if not isinstance(version, str) or not version:
        return
    current = active_configuration()
    if current is not None and current.version == version and not desired.get("test_requested"):
        return
    endpoint, model, language = desired.get("endpoint"), desired.get("model"), desired.get("language")
    if not all(isinstance(item, str) for item in (endpoint, model, language)):
        raise ValueError("invalid transcription configuration from server")
    reachability.set_target(endpoint, model)
    error = test_configuration(endpoint, model)
    result = client.get_httpx_client().post(
        "/api/worker/transcription-config/status",
        json={"version": version, "active": error is None, "error_code": error or ""},
    )
    result.raise_for_status()
    if error is None:
        activate(ActiveConfiguration(version, endpoint, model, language))


# Reachability is probed every 60 s, off the presence thread, and reported in each pulse (issue #212).
REACHABILITY_INTERVAL = 60.0
REACHABILITY_TIMEOUT = 5.0
BODY_EXCERPT_LIMIT = 200
_CONTROL = re.compile(r"[\x00-\x1f\x7f]+")


def _public_url(url: str) -> str:
    """Scheme, host, port and path only: no userinfo, no query, no fragment."""
    parts = urlsplit(url)
    host = parts.hostname or ""
    if ":" in host:
        host = f"[{host}]"
    netloc = host + (f":{parts.port}" if parts.port else "")
    return urlunsplit((parts.scheme, netloc, parts.path, "", ""))


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def _check(endpoint: str, timeout: float) -> tuple[bool, dict | None, int]:
    """Return (reachable, failure details or None, latency in ms) for GET /v1/models."""
    parts = urlsplit(endpoint)
    url = urlunsplit((parts.scheme, parts.netloc, parts.path.rstrip("/") + "/v1/models", parts.query, ""))
    started = time.monotonic()

    def latency() -> int:
        return int(round((time.monotonic() - started) * 1000))

    def failure(error_type: str, status: int | None = None, excerpt: str = "") -> dict:
        return {
            "failed_at": _utc_now(),
            "method": "GET",
            "url": _public_url(url),
            "status_code": status,
            "latency_ms": latency(),
            "error_type": error_type,
            "body_excerpt": excerpt,
        }

    try:
        response = httpx.get(url, timeout=timeout)
    except httpx.TimeoutException:
        return False, failure("timeout"), latency()
    except httpx.HTTPError:
        return False, failure("connection"), latency()
    excerpt = _CONTROL.sub(" ", response.text)[:BODY_EXCERPT_LIMIT]
    if not 200 <= response.status_code < 300:
        return False, failure("http_status", response.status_code, excerpt), latency()
    try:
        payload = response.json()
    except ValueError:
        return False, failure("invalid_body", response.status_code, excerpt), latency()
    if not isinstance(payload, dict) or not isinstance(payload.get("data"), list):
        return False, failure("invalid_body", response.status_code, excerpt), latency()
    return True, None, latency()


class ReachabilityMonitor:
    """Tracks whether the transcription server answers; never blocks the presence pulse.

    `run` executes in its own thread. The last result, and the last failure for the current
    target, are read by the presence thread through `snapshot`."""

    def __init__(self, interval: float = REACHABILITY_INTERVAL, timeout: float = REACHABILITY_TIMEOUT) -> None:
        self.interval = interval
        self.timeout = timeout
        self._lock = threading.Lock()
        self._target: tuple[str, str] | None = None
        self._kick = False
        self._result: dict | None = None
        self._last_failure: dict | None = None

    def set_target(self, endpoint: str, model: str) -> None:
        with self._lock:
            if self._target != (endpoint, model):
                self._target = (endpoint, model)
                self._last_failure = None
                self._result = None
                self._kick = True

    def clear_target(self) -> None:
        with self._lock:
            self._target = None
            self._result = None
            self._last_failure = None

    def probe_once(self) -> None:
        with self._lock:
            target = self._target
            self._kick = False
        if target is None:
            return
        reachable, failure, latency_ms = _check(target[0], self.timeout)
        with self._lock:
            if self._target != target:
                return  # The target changed while probing; the next probe reports the new one.
            if failure is not None:
                self._last_failure = failure
            self._result = {"checked_at": _utc_now(), "reachable": reachable, "latency_ms": latency_ms}

    def snapshot(self) -> dict | None:
        with self._lock:
            if self._result is None:
                return None
            payload = dict(self._result)
            payload["last_failure"] = dict(self._last_failure) if self._last_failure else None
            return payload

    def run(self, stop: threading.Event) -> None:
        due = 0.0
        while not stop.is_set():
            with self._lock:
                has_target = self._target is not None
                kicked = self._kick
            if has_target and (kicked or time.monotonic() >= due):
                self.probe_once()
                due = time.monotonic() + self.interval
            stop.wait(0.02)


reachability = ReachabilityMonitor()
