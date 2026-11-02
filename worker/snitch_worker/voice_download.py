"""Download subprocess for one voice model: the only code that reads the Hugging Face token.

    python -m snitch_worker.voice_download <repo> <revision> <target>

The token is read from the file named by ``SNITCH_HF_TOKEN_FILE`` (never from the environment,
never from the API) and handed to the Hugging Face library only. Output is JSON lines on stdout:
progress (``received``/``total`` bytes), then ``done`` or a stable ``error`` code. Library
messages, which may echo the token, are never printed.
"""

from __future__ import annotations

import json
import os
import sys
from collections.abc import Callable, Mapping
from typing import Any

TOKEN_FILE_ENV = "SNITCH_HF_TOKEN_FILE"
# The only places the token may travel: the Hugging Face variables are never passed on, and the file path
# only to the download subprocess.
TOKEN_ENV_NAMES = frozenset({"HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"})


def without_token(environ: Mapping[str, str], *, keep_file: bool = False) -> dict[str, str]:
    """A copy of the environment without the token variables, and without the token file path unless kept."""
    dropped = TOKEN_ENV_NAMES if keep_file else TOKEN_ENV_NAMES | {TOKEN_FILE_ENV}
    return {key: value for key, value in environ.items() if key not in dropped}

Snapshot = Callable[[str, str, str, str, Callable[[int, int], None]], None]
Sizer = Callable[[str, str, str], int]
Emit = Callable[[str], None]


def _read_token(environ: Mapping[str, str]) -> str:
    path = environ.get(TOKEN_FILE_ENV, "").strip()
    if not path:
        return ""
    try:
        with open(path, encoding="utf-8") as handle:
            return handle.read().strip()
    except OSError:
        return ""


def _cause(error: BaseException) -> str:
    names = {cls.__name__ for cls in type(error).__mro__}
    status = getattr(getattr(error, "response", None), "status_code", None)
    if names & {"GatedRepoError", "RepositoryNotFoundError", "RevisionNotFoundError"} or status in (401, 403, 404):
        return "hf_access_denied"
    if isinstance(error, OSError) and getattr(error, "errno", None) == 28:
        return "disk_full"
    return "runtime_incompatible"


def run(repo: str, revision: str, target: str, environ: Mapping[str, str], snapshot: Snapshot, size_of: Sizer, emit: Emit) -> int:
    def say(event: dict[str, Any]) -> None:
        emit(json.dumps(event))

    token = _read_token(environ)
    if not token:
        say({"error": "hf_token_missing"})
        return 1
    try:
        total = size_of(repo, revision, token)

        def on_progress(received: int, size: int) -> None:
            say({"received": received, "total": size or total})

        snapshot(repo, revision, target, token, on_progress)
    except BaseException as error:  # noqa: BLE001 - library messages may carry the token
        if isinstance(error, (KeyboardInterrupt, SystemExit)):
            raise
        say({"error": _cause(error)})
        return 1
    say({"done": True})
    return 0


def _hf_size(repo: str, revision: str, token: str) -> int:  # pragma: no cover - needs network
    from huggingface_hub import HfApi

    info = HfApi().model_info(repo, revision=revision, token=token, files_metadata=True)
    return sum(item.size or 0 for item in info.siblings or [])


def _hf_snapshot(repo: str, revision: str, target: str, token: str, on_progress: Callable[[int, int], None]) -> None:  # pragma: no cover - needs network
    from huggingface_hub import snapshot_download
    from tqdm.auto import tqdm

    total = _hf_size(repo, revision, token)
    done = [0]

    class Bar(tqdm):
        def update(self, n: float | None = 1) -> bool | None:
            result = super().update(n)
            if getattr(self, "unit", "") == "B" and n:
                done[0] += int(n)
                on_progress(min(done[0], total), total)
            return result

    snapshot_download(repo_id=repo, revision=revision, local_dir=target, token=token, tqdm_class=Bar)


def main(argv: list[str]) -> int:  # pragma: no cover - needs network
    if len(argv) != 3:
        return 2
    return run(argv[0], argv[1], argv[2], os.environ, _hf_snapshot, _hf_size, lambda line: print(line, flush=True))


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main(sys.argv[1:]))
