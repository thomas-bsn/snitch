"""Start of the Worker image, run before the worker.

    python -m snitch_worker.bootstrap

It downloads nothing and prepares no voice model: a Worker starts without any, and only acts on the
orders Snitch gives it (ADR 0038, ``snitch_worker.model_orders``). What remains is what must happen
before the worker process: keep the Hugging Face token an operator gives to the ``store-token``
command (ADR 0027), and start the worker without ``HF_TOKEN`` and always with ``HF_HUB_OFFLINE=1``.
"""

from __future__ import annotations

import os
import sys
from typing import Any, Callable, Mapping

from snitch_worker import log as worker_log
from snitch_worker import hf_token_sync, revocation, voice_download

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


def main(environ: Mapping[str, str] | None = None, exec_worker: Callable[[dict[str, str]], None] | None = None) -> None:
    env = dict(os.environ if environ is None else environ)
    worker_log.configure(env.get("LOG_LEVEL"))
    # Revoked or not, nothing is downloaded here: the runner alone decides whether to stay inert.
    (exec_worker or _exec_worker)(worker_environment(env))


def cli(
    argv: list[str],
    environ: Mapping[str, str] | None = None,
    stdin: Any = None,
    stdout: Any = None,
    exec_worker: Callable[[dict[str, str]], None] | None = None,
) -> int:
    """``store-token`` keeps a token read on standard input; any other command starts the worker."""
    env = dict(os.environ if environ is None else environ)
    if argv[:1] == ["store-token"]:
        return store_token(env, stdin or sys.stdin, stdout or sys.stdout)
    main(env, exec_worker)
    return 0


if __name__ == "__main__":
    sys.exit(cli(sys.argv[1:]))
