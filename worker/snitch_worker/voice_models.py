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
    return bool(_REPO.fullmatch(repo)) and bool(_REVISION.fullmatch(revision))


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


# Where the bootstrap wrote each type before models lived side by side: manifest key, path variable, default path.
_LEGACY = {
    "diarization": ("diarization", voice_manifest.DIARIZATION_PATH_ENV, voice_manifest.DEFAULT_DIARIZATION_PATH),
    "fingerprint": ("embedding", voice_manifest.EMBEDDING_PATH_ENV, voice_manifest.DEFAULT_EMBEDDING_PATH),
}


def kind_of(item: Mapping[str, Any]) -> str:
    """The type of an item; the diarization model when the server does not say."""
    return item["kind"] if item.get("kind") in (*_LEGACY, "vad") else "diarization"


def _legacy_path(environ: Mapping[str, str], repo: str, revision: str, kind: str = "diarization") -> str:
    if kind not in _LEGACY:
        return ""
    key, variable, default = _LEGACY[kind]
    entry = (voice_manifest.read_manifest(environ) or {}).get(key)
    if not isinstance(entry, dict) or entry.get("repo") != repo or entry.get("revision") != revision:
        return ""
    path = environ.get(variable, "").strip() or default
    try:
        return path if os.path.isdir(path) and os.listdir(path) else ""
    except OSError:
        return ""


def ready_path(environ: Mapping[str, str], repo: str, revision: str, kind: str) -> str:
    """Resolve a validated reference, including proofs made against legacy weights.

    Older installers wrote the proof in the catalog even when inference used the
    legacy directory. Keep those installations usable without copying or fetching.
    """
    if not is_ready(environ, repo, revision):
        return ""
    return _legacy_path(environ, repo, revision, kind) or model_path(environ, repo, revision)


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
_active_references: dict[str, dict[str, str]] = {}


def active_references() -> dict[str, dict[str, str]]:
    return {kind: dict(reference) for kind, reference in _active_references.items()}


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
        verify_fingerprint: Verify | None = None,
        install_vad: Callable[..., None] | None = None,
    ) -> None:
        self._env = environ
        self._download = download
        self._verify = verify
        # A fingerprint model proves itself with an embedding, never with a diarization.
        self._verify_fingerprint = verify_fingerprint or verify
        # Held models whose proof failed once in this process: not run again at each pull.
        self.failed_proofs: set[tuple[str, str]] = set()
        self._report = report
        self._interval = min_progress_interval
        # The pinned speech detector of the audio diagnostics (ADR 0048), installed by an order only.
        self._install_vad = install_vad

    def note_active(self, desired: Mapping[str, Any]) -> None:
        """Remember which repository Snitch holds as active: it only labels the declaration, nothing is fetched."""
        global _active_repo, _active_references
        # Only the diarization model labels the declaration (the fingerprint model is listed too, with its kind).
        models = [m for m in desired.get("models") or [] if isinstance(m, dict) and m.get("kind", "diarization") == "diarization"]
        _active_repo = next((m["repo"] for m in models if m.get("role") == "active" and isinstance(m.get("repo"), str)), "")
        _active_references = {
            kind_of(m): {"repo": m["repo"], "revision": m["revision"]}
            for m in desired.get("models") or []
            if isinstance(m, dict) and m.get("role") == "active"
            and isinstance(m.get("repo"), str) and isinstance(m.get("revision"), str)
            and valid_model(m["repo"], m["revision"])
        }

    def prepare(self, item: dict[str, Any], report: Report) -> None:
        """Prepare one model now, in the calling thread (an order runs on its own thread)."""
        self._prepare(item, report)

    def holds(self, item: Mapping[str, Any]) -> bool:
        """Whether this volume holds the model: a verified copy, or the bootstrap's copy of the pinned revision.

        A model that is not on the volume is never fetched here: only an order downloads."""
        repo, revision = item.get("repo"), item.get("revision")
        if not (isinstance(repo, str) and isinstance(revision, str) and valid_model(repo, revision)):
            return False
        kind = kind_of(item)
        if kind == "vad":
            return is_ready(self._env, repo, revision)
        return is_ready(self._env, repo, revision) or bool(_legacy_path(self._env, repo, revision, kind))

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
        legacy = _legacy_path(self._env, repo, revision, kind_of(item))
        if legacy:
            shutil.rmtree(legacy, ignore_errors=True)
            if os.path.exists(legacy):
                post({"model_id": model_id, "state": "failed", "cause_code": "models_volume_unwritable", "bytes_received": 0, "bytes_total": 0})
                return
        shutil.rmtree(path, ignore_errors=True)
        if os.path.exists(path):
            # Never claim a removal that did not happen: Snitch keeps the model and offers Retry.
            log.emit(logging.INFO, "voice_model_removal_failed", model_id=model_id, revision=revision)
            post({"model_id": model_id, "state": "failed", "cause_code": "models_volume_unwritable", "bytes_received": 0, "bytes_total": 0})
            return
        post({"model_id": model_id, "state": "removed", "cause_code": "", "bytes_received": 0, "bytes_total": 0})

    def _prepare_vad(self, repo: str, revision: str, target: str, say: Callable[..., None]) -> None:
        """Install the one pinned speech detector this order names: checksum, then a real CPU inference."""
        from snitch_worker import audio_diagnostic

        if repo != audio_diagnostic.REPO or revision != audio_diagnostic.REVISION:
            # Only the artifact the adapter is versioned for is executable: never an arbitrary TorchScript file.
            say("failed", "runtime_incompatible")
            return
        install = self._install_vad or audio_diagnostic.install
        say("downloading")
        try:
            install(target, download=True, report=lambda state: say(state) if state == "verifying" else None)
        except BaseException as error:  # noqa: BLE001 - only a closed cause leaves this function
            if isinstance(error, (KeyboardInterrupt, SystemExit)):
                raise
            cause = _vad_cause(error)
            log.emit(logging.INFO, "voice_model_failed", phase="vad", cause_code=cause, model_id=repo, revision=revision)
            say("failed", cause)
            return
        say("ready")

    def _prepare(self, item: dict[str, Any], post: Report) -> None:
        repo, revision, model_id = item.get("repo"), item.get("revision"), item.get("model_id")
        if not (isinstance(repo, str) and isinstance(revision, str) and isinstance(model_id, str) and valid_model(repo, revision)):
            return
        milestone = {"model_id": model_id, "revision": revision}

        def say(state: str, cause: str = "", received: int = 0, total: int = 0) -> None:
            post({"model_id": model_id, "state": state, "cause_code": cause, "bytes_received": received, "bytes_total": total})

        target = model_path(self._env, repo, revision)
        location = _legacy_path(self._env, repo, revision, kind_of(item)) or target
        runtime = item.get("runtime")
        was_ready = is_ready(self._env, repo, revision)
        intent = item.get("runtime_intent") or "install"
        if intent != "install" and (runtime is None or not was_ready):
            # An update or a rollback reuses weights already validated: it never downloads them.
            say("failed", "runtime_incompatible")
            return
        if runtime is not None:
            from snitch_worker import engine_runtime
            try:
                engine_runtime.validate(runtime, item.get("engine", ""), kind_of(item))
                if intent == "rollback":
                    # Going back reuses an installed version: a lost one is never fetched again here.
                    if not engine_runtime.installed(runtime, self._env):
                        say("failed", "runtime_incompatible")
                        return
                else:
                    engine_runtime.install(runtime, self._env, say)
                if was_ready and engine_runtime.model_binding(location) == runtime:
                    say("ready")
                    return
            except engine_runtime.RuntimeFailed as error:
                # Resource causes of an engine (RAM, VRAM, CUDA, platform) fold onto the model order's causes.
                say("failed", error.cause if error.cause in {"disk_full", "hf_network", "runtime_incompatible"} else "runtime_incompatible")
                return
        if was_ready and runtime is None:
            from snitch_worker import engine_runtime
            try:
                binding = engine_runtime.model_binding(location)
                if binding is not None and not engine_runtime.installed(binding, self._env):
                    say("failed", "runtime_incompatible")
                    return
            except engine_runtime.RuntimeFailed as error:
                say("failed", error.cause)
                return
            say("ready")
            return
        kind = kind_of(item)
        if kind == "vad":
            self._prepare_vad(repo, revision, target, say)
            return
        verify = self._verify_fingerprint if kind == "fingerprint" else self._verify
        legacy = _legacy_path(self._env, repo, revision, kind)
        location = legacy or target
        if not legacy and not was_ready:
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
            if runtime is None:
                verify(location, inference_environment(self._env))
            else:
                from snitch_worker import voice_runtime
                voice_runtime.call(runtime, location, self._env, operation="verify")
                engine_runtime.bind(location, runtime)
            os.makedirs(target, exist_ok=True)
            with open(os.path.join(target, READY_MARKER), "w", encoding="utf-8") as handle:
                handle.write(revision)
        except BaseException as error:  # noqa: BLE001
            if isinstance(error, (KeyboardInterrupt, SystemExit)):
                raise
            cause = _cause_of_verification(error)
            log.emit(logging.INFO, "voice_model_failed", phase="verification", cause_code=cause, refusal=getattr(error, "detail", None), **milestone)
            say("failed", cause)
            return
        log.emit(logging.INFO, "voice_model_validated", **milestone)
        if item.get("role") == "active":
            log.emit(logging.INFO, "voice_model_activated", **milestone)
        say("ready")


def _vad_cause(error: BaseException) -> str:
    """A closed cause for a failed detector installation (the voice vocabulary is reused, ADR 0048)."""
    from urllib.error import URLError

    cause = getattr(error, "cause", "")
    if cause in ("model_checksum_mismatch", "runtime_incompatible"):
        return "runtime_incompatible"
    if cause == "model_missing" or isinstance(error, (URLError, TimeoutError, ConnectionError)):
        return "hf_network"
    if isinstance(error, OSError) and getattr(error, "errno", None) == 28:
        return "disk_full"
    if isinstance(error, PermissionError):
        return "models_volume_unwritable"
    return "inference_failed"


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
    _reconcile_held(desired, manager, http)


def _reconcile_held(desired: Mapping[str, Any], manager: ModelManager, http: Any) -> None:
    """Report the models this volume already holds and Snitch does not know as ready.

    A model installed before orders existed (the bootstrap's copy) was never reported, so Snitch showed it as
    not installed and dispatched nothing to this Worker. It is proven once like any model (loaded and run on
    synthetic input, the proof is the marker on the volume), then reported. Nothing is downloaded and nothing is
    removed; a model with an order in flight is left to that order."""
    known = desired.get("held")
    if not isinstance(known, list):
        return  # a server that does not say what it knows: nothing to compare with
    ordered = {o.get("model_id") for o in desired.get("orders") or [] if isinstance(o, dict)}
    for item in desired.get("models") or []:
        if not isinstance(item, dict) or item.get("model_id") in ordered:
            continue
        if item.get("model_id") in known:
            # A ready declaration must not outlive its isolated runtime. Checking the
            # local proof costs no inference and never provisions dependencies.
            from snitch_worker import engine_runtime
            location = _legacy_path(manager._env, item.get("repo", ""), item.get("revision", ""), kind_of(item)) or model_path(manager._env, item.get("repo", ""), item.get("revision", ""))
            try:
                binding = engine_runtime.model_binding(location)
                unavailable = binding is not None and not engine_runtime.installed(binding, manager._env)
            except engine_runtime.RuntimeFailed:
                unavailable = True
            if unavailable:
                http.post("/api/worker/voice-models/status", json={"model_id": item["model_id"], "revision": item["revision"], "state": "failed", "cause_code": "runtime_incompatible", "bytes_received": 0, "bytes_total": 0, "occupied_bytes": 0}).raise_for_status()
            continue
        proof = (str(item.get("model_id")), str(item.get("revision")))
        if proof in manager.failed_proofs or not manager.holds(item):
            continue
        reports: list[dict[str, Any]] = []
        manager.prepare(item, reports.append)
        final = reports[-1] if reports else None
        if final is None or final["state"] not in ("ready", "failed"):
            continue
        if final["state"] == "failed":
            manager.failed_proofs.add(proof)
        body = {**final, "revision": item["revision"], "occupied_bytes": 0}
        http.post("/api/worker/voice-models/status", json=body).raise_for_status()


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

    from snitch_worker import engine_runtime, voice_runtime
    runtime = engine_runtime.model_binding(path)
    if runtime is not None:
        voice_runtime.call(runtime, path, environ, operation="verify")
        return
    engine = engine_of(path)
    provider = (NemoProvider if engine == "nemo" else PyannoteProvider)(environ=dict(environ) | {voice_manifest.DIARIZATION_PATH_ENV: path})
    voice_selftest.diarize_synthetic(provider, engine)


def _fingerprint_verify(path: str, environ: Mapping[str, str]) -> None:  # pragma: no cover - needs models
    """Load the embedding model held at ``path`` and compute a valid fingerprint of a synthetic tone."""
    from snitch_worker import voice_selftest
    from snitch_worker.voice_pyannote import PyannoteProvider

    from snitch_worker import engine_runtime, voice_runtime
    runtime = engine_runtime.model_binding(path)
    if runtime is not None:
        voice_runtime.call(runtime, path, environ, operation="verify")
        return
    provider = PyannoteProvider(environ=dict(environ) | {voice_manifest.EMBEDDING_PATH_ENV: path})
    voice_selftest.embed_synthetic(provider)


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

        _manager = ModelManager(os.environ, download, _inference_verify, verify_fingerprint=_fingerprint_verify)
    return _manager
