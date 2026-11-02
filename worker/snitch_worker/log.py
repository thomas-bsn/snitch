"""Structured JSON logs with the same schema as the server (timestamp, level, service, event, context).

``LOG_LEVEL`` is ERROR, WARN, INFO (default) or DEBUG. Keys that could carry a secret or private content
are masked at every level, and exception text is never written: only a code or the exception class.
"""

from __future__ import annotations

import json
import logging
import os
import sys
from datetime import datetime, timezone
from typing import Any, TextIO

SERVICE = "worker"
LOGGER_NAME = "snitch_worker"
MASKED = "[masked]"
FORBIDDEN = ("authorization", "token", "cookie", "password", "secret", "signed_url", "transcript", "embedding", "fingerprint", "api_key", "apikey", "bearer", "credential")
# Keys that carry free error text: the value is never written, only a code and a category (see failure()).
RAW_ERROR_KEYS = ("error", "err", "reason", "exception", "detail")
_LEVELS = {"ERROR": logging.ERROR, "WARN": logging.WARNING, "WARNING": logging.WARNING, "INFO": logging.INFO, "DEBUG": logging.DEBUG}


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
    if isinstance(value, (str, int, float, bool)) or value is None:
        return value
    return str(value)


class JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "timestamp": datetime.fromtimestamp(record.created, timezone.utc).isoformat(timespec="milliseconds"),
            "level": "WARN" if record.levelno == logging.WARNING else record.levelname,
            "service": SERVICE,
            "event": record.getMessage(),
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
