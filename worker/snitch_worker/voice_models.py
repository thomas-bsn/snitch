"""Voice model manager: prepares the one model an order hands it and reports what the Worker really holds.

Nothing here starts by itself: the manager downloads only when ``prepare`` is called for an order
(``snitch_worker.model_orders``, ADR 0038), never at start-up, on reconnection or from the list of
models Snitch shows. What it keeps is on the volume (the verified copy and its marker) and in the
persisted orders, so a restart repeats nothing. For a model it downloads, loads and runs a synthetic
inference, and reports downloading, verifying, ready or failed with a stable cause. A downloaded model
that was not verified is never ready.

Several models live side by side under ``<models volume>/catalog/<repo>/<revision>``. The legacy
layout written by the bootstrap (``/models/diarization`` plus its schema 1 marker) stays readable.

The Hugging Face token is read only by the download subprocess (``snitch_worker.voice_download``)
from the file named by ``SNITCH_HF_TOKEN_FILE``. This module never reads it, and the inference
environment built here never carries it. Standard library only.
"""

from __future__ import annotations

import json
import logging
import os
import re
import shutil
import subprocess
import sys
import time
from collections.abc import Callable, Mapping
from typing import Any

from snitch_worker import log, voice_manifest
from snitch_worker import voice_download
from snitch_worker.voice_download import TOKEN_FILE_ENV, without_token

logger = logging.getLogger(__name__)

READY_MARKER = ".snitch-ready"
_REPO = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,95}/[A-Za-z0-9][A-Za-z0-9._-]{0,95}$")
_REVISION = re.compile(r"^[0-9a-f]{40}$")

Progress = Callable[[int, int], None]
Download = Callable[[str, str, str, Progress], None]
Verify = Callable[[str, Mapping[str, str]], None]
Report = Callable[[dict[str, Any]], None]


class DownloadFailed(Exception):
    """A download that failed for a reason with a stable code."""

    def __init__(self, cause: str) -> None:
        super().__init__(cause)
        self.cause = cause


def valid_model(repo: str, revision: str) -> bool:
    return bool(_REPO.match(repo)) and bool(_REVISION.match(revision))


def models_root(environ: Mapping[str, str]) -> str:
    return os.path.join(os.path.dirname(voice_manifest.manifest_path(environ)), "catalog")


def model_path(environ: Mapping[str, str], repo: str, revision: str) -> str:
    return os.path.join(models_root(environ), repo.replace("/", "__"), revision)


def is_ready(environ: Mapping[str, str], repo: str, revision: str) -> bool:
    return valid_model(repo, revision) and os.path.exists(os.path.join(model_path(environ, repo, revision), READY_MARKER))


def ready_revision(environ: Mapping[str, str], repo: str) -> str:
    """The revision of a verified copy of ``repo`` on this volume, or an empty string."""
    directory = os.path.join(models_root(environ), repo.replace("/", "__"))
    try:
        for revision in sorted(os.listdir(directory)):
            if is_ready(environ, repo, revision):
                return revision
    except OSError:
        pass
    return ""


def engine_of(path: str) -> str:
    """The engine that runs the model held at ``path``: a ``.nemo`` archive means NeMo, else pyannote."""
    try:
        return "nemo" if any(name.endswith(".nemo") for name in os.listdir(path)) else "pyannote"
    except OSError:
        return "pyannote"


def inference_environment(environ: Mapping[str, str]) -> dict[str, str]:
    """What the inference process may see: never the token, nor the path of its secret file."""
    env = without_token(environ)
    env["HF_HUB_OFFLINE"] = "1"
    env["TRANSFORMERS_OFFLINE"] = "1"
    return env


def _legacy_path(environ: Mapping[str, str], repo: str, revision: str) -> str:
    entry = (voice_manifest.read_manifest(environ) or {}).get("diarization")
    if not isinstance(entry, dict) or entry.get("repo") != repo or entry.get("revision") != revision:
        return ""
    path = environ.get(voice_manifest.DIARIZATION_PATH_ENV, "").strip() or voice_manifest.DEFAULT_DIARIZATION_PATH
    try:
        return path if os.path.isdir(path) and os.listdir(path) else ""
    except OSError:
        return ""


# Closed vocabulary of the download subprocess (stdout, one JSON line per event): the launcher keeps
# nothing else, so a library message can never travel further.
DOWNLOAD_CAUSES = frozenset({
    "hf_token_missing", "hf_access_denied", "hf_auth_required", "hf_not_found", "hf_network",
    "models_volume_unwritable", "hf_cache_error", "disk_full", "runtime_incompatible",
})
PROGRESS_STEPS = (25, 50, 75, 100)


def _cause_of_download(error: BaseException, environ: Mapping[str, str] | None = None) -> str:
    if isinstance(error, DownloadFailed):
        return error.cause
    return voice_download._cause(error, os.environ if environ is None else environ)


def read_download_output(lines: Any, progress: Progress) -> str:
    """Forward the progress of the download subprocess and return its failure code (never its text)."""
    cause = "runtime_incompatible"
    for line in lines:
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if not isinstance(event, dict):
            continue
        if "received" in event:
            progress(int(event["received"]), int(event["total"]))
        elif event.get("error") in DOWNLOAD_CAUSES:
            cause = event["error"]
    return cause


def _cause_of_verification(error: BaseException) -> str:
    reason = getattr(error, "reason", "")
    if reason == "inference_failed":
        return "inference_failed"
    if reason:
        return "runtime_incompatible"
    if isinstance(error, OSError) and getattr(error, "errno", None) == 28:
        return "disk_full"
    return "inference_failed"


_active_repo = ""


def active_repo() -> str:
    """The repository of the active diarization model, as the server last said."""
    return _active_repo


class ModelManager:
    def __init__(
        self,
        environ: Mapping[str, str],
        download: Download,
        verify: Verify,
        report: Report | None = None,
        min_progress_interval: float = 1.0,
    ) -> None:
        self._env = environ
        self._download = download
        self._verify = verify
        self._report = report
        self._interval = min_progress_interval

    def note_active(self, desired: Mapping[str, Any]) -> None:
        """Remember which repository Snitch holds as active: it only labels the declaration, nothing is fetched."""
        global _active_repo
        models = [m for m in desired.get("models") or [] if isinstance(m, dict)]
        _active_repo = next((m["repo"] for m in models if m.get("role") == "active" and isinstance(m.get("repo"), str)), "")

    def prepare(self, item: dict[str, Any], report: Report) -> None:
        """Prepare one model now, in the calling thread (an order runs on its own thread)."""
        self._prepare(item, report)

    def remove(self, items: list[dict[str, Any]], report: Report | None = None) -> None:
        post = report or self._report or (lambda _payload: None)
        for item in items:
            if isinstance(item, dict):
                self._remove(item, post)

    def _remove(self, item: dict[str, Any], post: Report) -> None:
        repo, revision, model_id = item.get("repo"), item.get("revision"), item.get("model_id")
        if not (isinstance(repo, str) and isinstance(revision, str) and isinstance(model_id, str) and valid_model(repo, revision)):
            return
        path = model_path(self._env, repo, revision)
        shutil.rmtree(path, ignore_errors=True)
        if os.path.exists(path):
            # Never claim a removal that did not happen: Snitch keeps the model and offers Retry.
            log.emit(logging.INFO, "voice_model_removal_failed", model_id=model_id, revision=revision)
            post({"model_id": model_id, "state": "failed", "cause_code": "models_volume_unwritable", "bytes_received": 0, "bytes_total": 0})
            return
        post({"model_id": model_id, "state": "removed", "cause_code": "", "bytes_received": 0, "bytes_total": 0})

    def _prepare(self, item: dict[str, Any], post: Report) -> None:
        repo, revision, model_id = item.get("repo"), item.get("revision"), item.get("model_id")
        if not (isinstance(repo, str) and isinstance(revision, str) and isinstance(model_id, str) and valid_model(repo, revision)):
            return
        milestone = {"model_id": model_id, "revision": revision}

        def say(state: str, cause: str = "", received: int = 0, total: int = 0) -> None:
            post({"model_id": model_id, "state": state, "cause_code": cause, "bytes_received": received, "bytes_total": total})

        target = model_path(self._env, repo, revision)
        if is_ready(self._env, repo, revision):
            say("ready")
            return
        legacy = _legacy_path(self._env, repo, revision)
        location = legacy or target
        if not legacy:
            say("downloading")
            last = [0.0]
            reached = [0]
            resuming = os.path.isdir(target) and bool(os.listdir(target))
            log.emit(logging.INFO, "voice_model_download_resumed" if resuming else "voice_model_download_started", **milestone)

            def progress(received: int, total: int) -> None:
                if total > 0:
                    percent = min(100, received * 100 // total)
                    step = max((s for s in PROGRESS_STEPS if s <= percent), default=0)
                    if step > reached[0]:
                        reached[0] = step
                        log.emit(logging.DEBUG, "voice_model_download_progress", percent=step, **milestone)
                now = time.monotonic()
                if received < total and now - last[0] < self._interval:
                    return
                last[0] = now
                say("downloading", received=received, total=total)

            try:
                os.makedirs(target, exist_ok=True)
                self._download(repo, revision, target, progress)
            except BaseException as error:  # noqa: BLE001 - never let a library message through
                if isinstance(error, (KeyboardInterrupt, SystemExit)):
                    raise
                cause = _cause_of_download(error, self._env)
                log.emit(logging.INFO, "voice_model_failed", phase="download", cause_code=cause, **milestone)
                say("failed", cause)
                return
        say("verifying")
        try:
            self._verify(location, inference_environment(self._env))
            with open(os.path.join(target, READY_MARKER), "w", encoding="utf-8") as handle:
                handle.write(revision)
        except BaseException as error:  # noqa: BLE001
            if isinstance(error, (KeyboardInterrupt, SystemExit)):
                raise
            cause = _cause_of_verification(error)
            log.emit(logging.INFO, "voice_model_failed", phase="verification", cause_code=cause, **milestone)
            say("failed", cause)
            return
        log.emit(logging.INFO, "voice_model_validated", **milestone)
        if item.get("role") == "active":
            log.emit(logging.INFO, "voice_model_activated", **milestone)
        say("ready")


def sync_voice_models(client: Any, manager: ModelManager) -> None:
    """Pull what Snitch holds for this Worker: only the active model is noted (it labels the declaration).

    Nothing is downloaded and nothing is removed here: models are removed only on a remove order (``model_orders``)."""
    http = client.get_httpx_client()
    response = http.get("/api/worker/voice-models")
    if response.status_code == 404:
        return  # A worker binary may briefly talk to a server predating this contract.
    response.raise_for_status()
    desired = response.json()
    if not isinstance(desired, dict):
        raise ValueError("invalid voice models from server")

    manager.note_active(desired)


def download_environment(token_file: str) -> dict[str, str]:
    """The download subprocess alone receives the path of the token file, explicitly."""
    env = without_token(os.environ)
    env["HF_HUB_OFFLINE"] = "0"
    if token_file:
        env[TOKEN_FILE_ENV] = token_file
    return env


def _subprocess_download(repo: str, revision: str, target: str, progress: Progress, token_file: str = "") -> None:  # pragma: no cover - needs network
    """Run the download in a subprocess: it alone reads the token file."""
    env = download_environment(token_file)
    process = subprocess.Popen(
        [sys.executable, "-m", "snitch_worker.voice_download", repo, revision, target],
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        env=env,
    )
    assert process.stdout is not None
    cause = read_download_output(process.stdout, progress)
    if process.wait() != 0:
        raise DownloadFailed(cause)


def _inference_verify(path: str, environ: Mapping[str, str]) -> None:  # pragma: no cover - needs models
    from snitch_worker import voice_selftest
    from snitch_worker.voice_nemo import NemoProvider
    from snitch_worker.voice_pyannote import PyannoteProvider

    engine = engine_of(path)
    provider = (NemoProvider if engine == "nemo" else PyannoteProvider)(environ=dict(environ) | {voice_manifest.DIARIZATION_PATH_ENV: path})
    voice_selftest.diarize_synthetic(provider, engine)


_manager: ModelManager | None = None


def default_manager() -> ModelManager:  # pragma: no cover - wires the real download and inference
    global _manager
    if _manager is None:
        # The manager is the only reader of the token file path: it leaves the process environment, so nothing
        # started later from this process (inference, tools) can inherit it.
        token_file = voice_download.default_token_file(os.environ)
        os.environ.pop(TOKEN_FILE_ENV, None)

        def download(repo: str, revision: str, target: str, progress: Progress) -> None:
            _subprocess_download(repo, revision, target, progress, token_file)

        _manager = ModelManager(os.environ, download, _inference_verify)
    return _manager
