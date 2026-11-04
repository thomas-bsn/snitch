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

import logging
import os
import sys
from typing import Any, Callable, Mapping

from snitch_worker import log as worker_log
from snitch_worker import hf_token_sync, revocation, voice_download, voice_manifest

DIARIZATION_REPO = "pyannote/speaker-diarization-community-1"
EMBEDDING_REPO = "pyannote/wespeaker-voxceleb-resnet34-LM"
REVISION_ENV = "SNITCH_VOICE_MODEL_REVISION"
LOG_PREFIX = "[snitch-bootstrap] "

Downloader = Callable[..., Any]
RevisionResolver = Callable[[str, str, str], str]


def _truthy(value: str | None) -> bool:
    return (value or "").strip().lower() in {"1", "true", "yes", "on"}


# Closed vocabulary shared with the installer launcher. The fine Hugging Face causes are printed on
# standard output by ``provision``; the launcher maps them onto the installation-event vocabulary.
CAUSE_CODES = (
    "hf_token_invalid",
    "hf_gated_denied",
    "hf_repo_invalid",
    "network_unreachable",
    "volume_not_writable",
    "hf_cache_not_writable",
    "gpu_unavailable",
    "unknown",
)
CAUSE_TEXT = {
    "hf_token_invalid": "Hugging Face refused the token (missing, mistyped or revoked): create a read token in your Hugging Face settings and rerun",
    "hf_gated_denied": "the account that owns the token has not accepted the terms of both models (gated access refused): accept them on huggingface.co with that account and rerun",
    "hf_repo_invalid": "a model repository or its revision was not found on Hugging Face: the Snitch version may not match, report it to the administrator",
    "network_unreachable": "network unreachable (huggingface.co could not be contacted)",
    "volume_not_writable": "models directory is not writable (mount it read-write for the first download)",
    "hf_cache_not_writable": "the Hugging Face/Xet cache is not writable (check the models volume ownership)",
    "gpu_unavailable": "no usable NVIDIA GPU for the voice models",
    "unknown": "unexpected bootstrap error",
}


class NoRevisionError(LookupError):
    """The Hub answered without a commit for the requested revision."""


_REPO_INVALID_NAMES = {"RepositoryNotFoundError", "RevisionNotFoundError", "EntryNotFoundError", "NoRevisionError"}


def cause_code(error: BaseException, environ: Mapping[str, str] | None = None) -> str:
    """A stable code. Never derived from, nor containing, the exception text (it may echo the token)."""
    names = {cls.__name__ for cls in type(error).__mro__}
    status = getattr(getattr(error, "response", None), "status_code", None)
    # GatedRepoError subclasses RepositoryNotFoundError: the gate is checked first.
    if "GatedRepoError" in names or status == 403:
        return "hf_gated_denied"
    if status == 401:
        return "hf_token_invalid"
    if names & _REPO_INVALID_NAMES or status == 404:
        return "hf_repo_invalid"
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


GPU_DEVICE_NODES = ("/dev/nvidiactl", "/dev/dxg")  # native Linux, WSL2


def _torch_cuda() -> bool | None:  # pragma: no cover - needs torch
    """True/False when PyTorch can tell whether CUDA is usable, None when it cannot be asked."""
    try:
        import torch  # noqa: PLC0415 - lazy: the bootstrap must run without the ML stack
    except Exception:  # noqa: BLE001
        return None
    try:
        return bool(torch.cuda.is_available())
    except Exception:  # noqa: BLE001
        return False


def _gpu_available(exists: Callable[[str], bool] = os.path.exists, torch_cuda: Callable[[], bool | None] = _torch_cuda) -> bool:
    """CUDA actually usable: PyTorch is the authority; device nodes (/dev/dxg on WSL2) only when it is absent."""
    verdict = torch_cuda()
    if verdict is not None:
        return verdict
    return any(exists(node) for node in GPU_DEVICE_NODES)


def _default_downloader(**kwargs: Any) -> Any:  # pragma: no cover - needs network
    from huggingface_hub import snapshot_download  # lazy: only an opted-in download needs it

    return snapshot_download(**kwargs)


def _default_resolver(repo_id: str, revision: str, token: str) -> str:  # pragma: no cover - needs network
    from huggingface_hub import HfApi

    sha = HfApi().model_info(repo_id, revision=revision or None, token=token).sha
    if not sha:
        raise NoRevisionError(repo_id)
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
        text = text.replace(token, "***") if token else text
        (log or (lambda line: worker_log.emit(logging.INFO, "bootstrap", message=line)))(text)

    if not _truthy(environ.get("SNITCH_VOICE_DOWNLOAD_ON_START")):
        say("voice model download is off (SNITCH_VOICE_DOWNLOAD_ON_START is not 1): nothing is downloaded")
        return "disabled"
    _keep_token(environ, token, say)
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


def _keep_token(environ: Mapping[str, str], token: str, say: Callable[[str], None]) -> None:
    """Keep the installation's token in the models volume, so Snitch can prepare other models later."""
    if not token:
        return
    if voice_download.persist_token(voice_download.default_token_file(environ), token):
        hf_token_sync.mark_local(voice_download.default_token_file(environ))
        say("Hugging Face token kept in the models volume for later model installs")
    else:
        say("Hugging Face token not kept: unexpected format or the models volume is not writable")


def store_token(environ: Mapping[str, str], stdin: Any, stdout: Any) -> int:
    """``store-token``: read a token from standard input (never argv or the environment) and keep it."""
    if revocation.is_revoked(revocation.models_dir(environ)):
        # A deleted Worker keeps nothing: its volume was wiped with the token file.
        print("cause=worker_revoked", file=stdout, flush=True)
        return 1
    token = stdin.readline().strip()
    if not voice_download.valid_token(token):
        print("cause=invalid_token", file=stdout, flush=True)
        return 1
    if not voice_download.persist_token(voice_download.default_token_file(environ), token):
        print("cause=volume_not_writable", file=stdout, flush=True)
        return 1
    hf_token_sync.mark_local(voice_download.default_token_file(environ))
    print("stored", file=stdout, flush=True)
    return 0


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
    worker_log.configure(env.get("LOG_LEVEL"))
    if revocation.is_revoked(revocation.models_dir(env)):
        # Deleted in Snitch: download nothing; the runner stays inert without any network call.
        (exec_worker or _exec_worker)(worker_environment(env))
        return
    try:
        run_bootstrap(env, downloader, resolve_revision, log)
    except Exception as error:  # noqa: BLE001 - the worker must start regardless
        worker_log.emit(logging.WARNING, "bootstrap_failed", cause=_cause(error), message="the worker starts anyway")
    (exec_worker or _exec_worker)(worker_environment(env))


def cli(
    argv: list[str],
    environ: Mapping[str, str] | None = None,
    downloader: Downloader | None = None,
    resolve_revision: RevisionResolver | None = None,
    log: Callable[[str], None] | None = None,
    stdin: Any = None,
    stdout: Any = None,
) -> int:
    """Launcher entry points. ``check``: 0 when valid models are present. ``provision``: explicit
    download; prints only ``cause=<code>`` on stdout and exits 1 when it failed."""
    env = dict(os.environ if environ is None else environ)
    if argv[:1] == ["check"]:
        return 0 if models_present(env) else 1
    if argv[:1] == ["store-token"]:
        return store_token(env, stdin or sys.stdin, stdout or sys.stdout)
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
