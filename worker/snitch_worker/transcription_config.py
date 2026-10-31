"""Worker-side validation and active transcription configuration."""

from __future__ import annotations

import threading
from dataclasses import dataclass
from urllib.parse import urlparse

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
    error = test_configuration(endpoint, model)
    result = client.get_httpx_client().post(
        "/api/worker/transcription-config/status",
        json={"version": version, "active": error is None, "error_code": error or ""},
    )
    result.raise_for_status()
    if error is None:
        activate(ActiveConfiguration(version, endpoint, model, language))
