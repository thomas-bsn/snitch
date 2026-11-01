"""Explicit, opt-in bootstrap run by the GPU image before the worker.

    python -m snitch_worker.bootstrap

Downloads the voice models into the models volume only when ALL of these hold:

* ``SNITCH_VOICE_DOWNLOAD_ON_START=1`` (explicit opt-in),
* ``SNITCH_VOICE_CONFIRM_TERMS=1`` (the operator accepted the model terms on Hugging Face),
* ``HF_TOKEN`` is set,
* the models are not already present and valid (marker ``.snitch-voice-models.json``).

Whatever happens, the worker then starts: a failed or skipped download only leaves the voice
steps ``provider_unavailable``. The worker process never receives ``HF_TOKEN`` and always runs
with ``HF_HUB_OFFLINE=1``. Messages are fixed strings: the token and library error texts are
never logged. Without the opt-in variables nothing is downloaded.
"""

from __future__ import annotations

import os
import sys
from typing import Any, Callable, Mapping

from snitch_worker import revocation, voice_manifest

DIARIZATION_REPO = "pyannote/speaker-diarization-community-1"
EMBEDDING_REPO = "pyannote/wespeaker-voxceleb-resnet34-LM"
REVISION_ENV = "SNITCH_VOICE_MODEL_REVISION"
LOG_PREFIX = "[snitch-bootstrap] "

Downloader = Callable[..., Any]
RevisionResolver = Callable[[str, str, str], str]


def _truthy(value: str | None) -> bool:
    return (value or "").strip().lower() in {"1", "true", "yes", "on"}


# Closed vocabulary shared with the installer launcher and the install-events contract.
CAUSE_CODES = ("hf_access_denied", "network_unreachable", "volume_not_writable", "hf_cache_not_writable", "gpu_unavailable", "unknown")
CAUSE_TEXT = {
    "hf_access_denied": "access refused: check that the token is valid and that the model terms were accepted with the account that owns it",
    "network_unreachable": "network unreachable (huggingface.co could not be contacted)",
    "volume_not_writable": "models directory is not writable (mount it read-write for the first download)",
    "hf_cache_not_writable": "the Hugging Face/Xet cache is not writable (check the models volume ownership)",
    "gpu_unavailable": "no usable NVIDIA GPU for the voice models",
    "unknown": "unexpected bootstrap error",
}


def cause_code(error: BaseException, environ: Mapping[str, str] | None = None) -> str:
    """A stable code. Never derived from, nor containing, the exception text (it may echo the token)."""
    names = {cls.__name__ for cls in type(error).__mro__}
    status = getattr(getattr(error, "response", None), "status_code", None)
    if "GatedRepoError" in names or "RepositoryNotFoundError" in names or status in (401, 403, 404):
        return "hf_access_denied"
    if names & {"ConnectionError", "Timeout", "TimeoutError", "OfflineModeIsEnabled", "ConnectTimeout", "ReadTimeout", "gaierror"}:
        return "network_unreachable"
    if names & {"PermissionError", "FileNotFoundError", "NotADirectoryError", "OSError"} and (
        names & {"PermissionError", "FileNotFoundError", "NotADirectoryError"} or getattr(error, "errno", None) in (1, 13, 28, 30)
    ):
        env = os.environ if environ is None else environ
        path = os.path.abspath(str(getattr(error, "filename", "") or ""))
        cache_roots = [env.get("HF_XET_CACHE", ""), env.get("HF_HOME", ""), env.get("XDG_CACHE_HOME", "")]
        if any(root and (path == root or path.startswith(root.rstrip("/") + "/")) for root in cache_roots):
            return "hf_cache_not_writable"
        return "volume_not_writable"
    return "unknown"


def _cause(error: BaseException, environ: Mapping[str, str] | None = None) -> str:
    return CAUSE_TEXT[cause_code(error, environ)]


def _paths(environ: Mapping[str, str]) -> tuple[str, str]:
    diarization = environ.get(voice_manifest.DIARIZATION_PATH_ENV, "").strip() or voice_manifest.DEFAULT_DIARIZATION_PATH
    embedding = environ.get(voice_manifest.EMBEDDING_PATH_ENV, "").strip() or voice_manifest.DEFAULT_EMBEDDING_PATH
    return diarization, embedding


def _non_empty_dir(path: str) -> bool:
    try:
        return os.path.isdir(path) and bool(os.listdir(path))
    except OSError:
        return False


def models_present(environ: Mapping[str, str]) -> bool:
    manifest = voice_manifest.read_manifest(environ)
    if not manifest:
        return False
    diarization_path, embedding_path = _paths(environ)
    pinned = environ.get(REVISION_ENV, "").strip()
    for key, repo, path in (("diarization", DIARIZATION_REPO, diarization_path), ("embedding", EMBEDDING_REPO, embedding_path)):
        entry = manifest.get(key)
        if not isinstance(entry, dict) or entry.get("repo") != repo or not entry.get("revision"):
            return False
        if pinned and entry["revision"] != pinned:
            return False
        if not _non_empty_dir(path):
            return False
    return True


def _gpu_available() -> bool:  # pragma: no cover - host dependent
    return os.path.exists("/dev/nvidiactl")


def _default_downloader(**kwargs: Any) -> Any:  # pragma: no cover - needs network
    from huggingface_hub import snapshot_download  # lazy: only an opted-in download needs it

    return snapshot_download(**kwargs)


def _default_resolver(repo_id: str, revision: str, token: str) -> str:  # pragma: no cover - needs network
    from huggingface_hub import HfApi

    sha = HfApi().model_info(repo_id, revision=revision or None, token=token).sha
    if not sha:
        raise ValueError("no revision")
    return sha


def run_bootstrap(
    environ: Mapping[str, str],
    downloader: Downloader | None = None,
    resolve_revision: RevisionResolver | None = None,
    log: Callable[[str], None] | None = None,
    report: dict[str, str] | None = None,
    gpu_available: Callable[[], bool] | None = None,
) -> str:
    """Return a status: disabled, present, terms_not_confirmed, token_missing, downloaded, failed.

    ``report`` receives ``cause_code`` (one of CAUSE_CODES) when the status is ``failed``.
    """
    report = report if report is not None else {}
    token = environ.get("HF_TOKEN", "").strip()

    def say(message: str) -> None:
        text = LOG_PREFIX + message
        (log or (lambda line: print(line, file=sys.stderr, flush=True)))(text.replace(token, "***") if token else text)

    if not _truthy(environ.get("SNITCH_VOICE_DOWNLOAD_ON_START")):
        say("voice model download is off (SNITCH_VOICE_DOWNLOAD_ON_START is not 1): nothing is downloaded")
        return "disabled"
    if models_present(environ):
        say("voice models already present: nothing to download")
        return "present"
    if not _truthy(environ.get("SNITCH_VOICE_CONFIRM_TERMS")):
        say("voice models missing, not downloaded: set SNITCH_VOICE_CONFIRM_TERMS=1 after accepting the model terms on Hugging Face; voice steps stay provider_unavailable")
        return "terms_not_confirmed"
    if not token:
        say("voice models missing, not downloaded: HF_TOKEN is not set; voice steps stay provider_unavailable")
        return "token_missing"

    if _truthy(environ.get("SNITCH_VOICE_REQUIRE_GPU")) and not (gpu_available or _gpu_available)():
        report["cause_code"] = "gpu_unavailable"
        say("no usable NVIDIA GPU: " + CAUSE_TEXT["gpu_unavailable"] + "; voice steps stay provider_unavailable")
        return "failed"
    # Hugging Face reads its offline switch at import: allow network for this process only.
    if downloader is None:
        os.environ["HF_HUB_OFFLINE"] = "0"
    download = downloader or _default_downloader
    resolve = resolve_revision or _default_resolver
    pinned = environ.get(REVISION_ENV, "").strip()
    diarization_path, embedding_path = _paths(environ)
    entries: dict[str, dict[str, str]] = {}
    say("downloading voice models (first start or models changed); this can take a few minutes")
    for key, repo, target in (("diarization", DIARIZATION_REPO, diarization_path), ("embedding", EMBEDDING_REPO, embedding_path)):
        try:
            revision = pinned or resolve(repo, "main", token)
            download(repo_id=repo, local_dir=target, token=token, revision=revision)
        except BaseException as error:  # noqa: BLE001 - never let a library message through
            if isinstance(error, (KeyboardInterrupt, SystemExit)):
                raise
            report["cause_code"] = cause_code(error, environ)
            say(f"download of {repo} failed: {CAUSE_TEXT[report['cause_code']]}; the worker starts anyway, voice steps stay provider_unavailable")
            return "failed"
        entries[key] = {"repo": repo, "revision": revision}
        say(f"downloaded {repo} at revision {revision}")
    try:
        voice_manifest.write_manifest(environ, entries)
    except OSError as error:
        report["cause_code"] = cause_code(error, environ)
        say(f"models downloaded but the version marker could not be written: {CAUSE_TEXT[report['cause_code']]}")
        return "failed"
    say("voice models ready; the worker will run offline")
    return "downloaded"


def worker_environment(environ: Mapping[str, str]) -> dict[str, str]:
    env = {key: value for key, value in environ.items() if key != "HF_TOKEN"}
    env["HF_HUB_OFFLINE"] = "1"
    env["TRANSFORMERS_OFFLINE"] = "1"
    return env


def _exec_worker(env: dict[str, str]) -> None:  # pragma: no cover - replaces the process
    os.execve(sys.executable, [sys.executable, "-m", "snitch_worker.runner"], env)


def main(
    environ: Mapping[str, str] | None = None,
    downloader: Downloader | None = None,
    resolve_revision: RevisionResolver | None = None,
    exec_worker: Callable[[dict[str, str]], None] | None = None,
    log: Callable[[str], None] | None = None,
) -> None:
    env = dict(os.environ if environ is None else environ)
    if revocation.is_revoked(revocation.models_dir(env)):
        # Deleted in Snitch: download nothing; the runner stays inert without any network call.
        (exec_worker or _exec_worker)(worker_environment(env))
        return
    try:
        run_bootstrap(env, downloader, resolve_revision, log)
    except Exception as error:  # noqa: BLE001 - the worker must start regardless
        print(LOG_PREFIX + "bootstrap failed: " + _cause(error) + "; the worker starts anyway", file=sys.stderr, flush=True)
    (exec_worker or _exec_worker)(worker_environment(env))


def cli(
    argv: list[str],
    environ: Mapping[str, str] | None = None,
    downloader: Downloader | None = None,
    resolve_revision: RevisionResolver | None = None,
    log: Callable[[str], None] | None = None,
) -> int:
    """Launcher entry points. ``check``: 0 when valid models are present. ``provision``: explicit
    download; prints only ``cause=<code>`` on stdout and exits 1 when it failed."""
    env = dict(os.environ if environ is None else environ)
    if argv[:1] == ["check"]:
        return 0 if models_present(env) else 1
    if argv[:1] == ["provision"]:
        report: dict[str, str] = {}
        try:
            status = run_bootstrap(env, downloader, resolve_revision, log, report)
        except Exception as error:  # noqa: BLE001
            report["cause_code"] = cause_code(error, env)
            status = "failed"
        if status == "failed":
            print("cause=" + report.get("cause_code", "unknown"), flush=True)
            return 1
        return 0 if status in {"downloaded", "present"} else 2
    main()
    return 0


if __name__ == "__main__":
    sys.exit(cli(sys.argv[1:]))
