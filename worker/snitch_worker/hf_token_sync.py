"""The central Hugging Face token, as seen by the Worker (#194, ADR 0027).

Each cycle the Worker compares the version of the central token with the one it recorded. Only a different version
makes it fetch the value, on its authenticated endpoint, and keep it in the token file of the models volume (the same
private file ``store-token`` and the installation write). Then it acknowledges what it did.

Provenance is recorded beside the token file in ``hf_token.meta``: ``central`` (a copy from Snitch, replaceable and
purgeable by Snitch), ``local`` (set by ``store-token``, the installation, or a token already present before this
feature, which is never replaced silently), or ``none``. The value is never logged, never put in an environment, an
argument or an error message, and never sent back.
"""

from __future__ import annotations

import json
import logging
import os
import tempfile
from typing import Any

from snitch_worker import log, voice_download

META_NAME = "hf_token.meta"
NONE_VERSION = "0"
ERROR_STORE = "hf_token_store_failed"


def meta_path(token_file: str) -> str:
    return os.path.join(os.path.dirname(token_file), META_NAME)


def read_meta(token_file: str) -> dict[str, Any] | None:
    try:
        with open(meta_path(token_file), encoding="utf-8") as handle:
            value = json.load(handle)
    except (OSError, ValueError):
        return None
    return value if isinstance(value, dict) else None


def write_meta(token_file: str, source: str, version: str, acked: bool) -> bool:
    """Record provenance and version atomically, with the same private permissions as the token file."""
    directory = os.path.dirname(token_file)
    payload = json.dumps({"source": source, "version": version, "acked": acked})
    try:
        os.makedirs(directory, mode=0o700, exist_ok=True)
        descriptor, temporary = tempfile.mkstemp(dir=directory, prefix=".hf_token_meta.")
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
                handle.write(payload)
            os.chmod(temporary, 0o600)
            os.replace(temporary, meta_path(token_file))
        except BaseException:
            try:
                os.unlink(temporary)
            except OSError:
                pass
            raise
    except OSError:
        return False
    return True


def mark_local(token_file: str) -> None:
    """Called when a local token is stored (store-token, installation): it is never replaced silently."""
    write_meta(token_file, "local", "", acked=True)


def current_source(token_file: str) -> str:
    """Where the token in the file comes from: local, central, or none (absent, or a central copy already removed)."""
    present = os.path.isfile(token_file)
    meta = read_meta(token_file)
    if meta is None:
        # Before this feature every token was local: a file without a record is a local token.
        return "local" if present else "none"
    source = meta.get("source")
    if source in ("local", "central") and present:
        return str(source)
    return "none"


def _remove_token(token_file: str) -> None:
    try:
        os.remove(token_file)
    except FileNotFoundError:
        pass


def _acknowledge(http: Any, version: str, outcome: str, source: str, error_code: str = "") -> None:
    response = http.post(
        "/api/worker/hf-token/ack",
        json={"version": version, "outcome": outcome, "source": source, "error_code": error_code},
    )
    response.raise_for_status()


def sync(client: Any, token_file: str) -> None:
    """Bring the central copy in line with the server's version; raises on transport or authentication errors."""
    http = client.get_httpx_client()
    response = http.get("/api/worker/hf-token/version")
    if response.status_code == 404:
        return  # A worker binary may briefly talk to a server predating this contract.
    response.raise_for_status()
    version = response.json().get("version")
    if not isinstance(version, str) or not version or version == NONE_VERSION:
        return  # No central token, or one the server cannot read: nothing is served, nothing is purged.
    meta = read_meta(token_file)
    if meta is not None and meta.get("version") == version and meta.get("acked") is True:
        return
    source = current_source(token_file)

    response = http.get("/api/worker/hf-token")
    response.raise_for_status()
    answer = response.json()
    if not isinstance(answer, dict) or answer.get("version") != version:
        return  # The version moved during the exchange: the next cycle asks again.
    token = answer.get("token")

    if token is None:
        if source == "local":
            _record(http, token_file, version, "kept_local", "local")
        elif source == "central":
            _remove_token(token_file)
            _record(http, token_file, version, "removed", "none")
        else:
            _record(http, token_file, version, "removed", "none")
        return

    if source == "local":
        _record(http, token_file, version, "kept_local", "local")
        return
    if not isinstance(token, str) or not voice_download.valid_token(token):
        _record(http, token_file, version, "failed", source)
        return
    # Provenance first: a copy interrupted midway is recorded as central, never as an unrecorded local token.
    if not write_meta(token_file, "central", version, acked=False) or not voice_download.persist_token(token_file, token):
        _record(http, token_file, version, "failed", source)
        return
    _record(http, token_file, version, "applied", "central")


def _record(http: Any, token_file: str, version: str, outcome: str, source: str) -> None:
    """Acknowledge an outcome; the provenance is recorded as acknowledged only once the server has it."""
    if outcome == "failed":
        _acknowledge(http, version, outcome, source, ERROR_STORE)
        log.emit(logging.WARNING, "hf_token_sync_failed", cause=ERROR_STORE)
        return
    _acknowledge(http, version, outcome, source)
    final_source = "none" if outcome == "removed" else source
    write_meta(token_file, final_source, version, acked=True)
    log.emit(logging.INFO, "hf_token_synced", outcome=outcome)
