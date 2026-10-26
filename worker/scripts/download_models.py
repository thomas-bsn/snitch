#!/usr/bin/env python3
"""Explicit, one-off download of the voice models into local directories.

Never called by the worker or the tests (the opt-in ``snitch_worker.bootstrap`` is the
automatic path, see docs/real-voice-worker.md). Run this one by hand on
a machine with network access, after accepting the model terms on Hugging Face
with the account that owns the token. The token is read from the HF_TOKEN
environment variable only, never from an argument, and is never printed.

    HF_TOKEN=... python scripts/download_models.py --confirm-terms \
        --diarization-dir /models/diarization --embedding-dir /models/embedding

Then point the worker at the directories (see docs/real-voice-worker.md):
SNITCH_VOICE_DIARIZATION_MODEL_PATH and SNITCH_VOICE_EMBEDDING_MODEL_PATH.
"""

from __future__ import annotations

import argparse
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
from snitch_worker.bootstrap import DIARIZATION_REPO, EMBEDDING_REPO  # noqa: E402 - one source for the repository names


def main(argv: list[str] | None = None, environ: dict[str, str] | None = None, downloader=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--diarization-dir", required=True)
    parser.add_argument("--embedding-dir", required=True)
    parser.add_argument("--diarization-repo", default=DIARIZATION_REPO)
    parser.add_argument("--embedding-repo", default=EMBEDDING_REPO)
    parser.add_argument("--revision", default=None, help="pin the same revision for both repositories (recommended)")
    parser.add_argument("--confirm-terms", action="store_true", help="I accepted the terms of both models on Hugging Face")
    args = parser.parse_args(argv)
    env = os.environ if environ is None else environ
    if not args.confirm_terms:
        print("refused: pass --confirm-terms after accepting the model terms", file=sys.stderr)
        return 2
    token = env.get("HF_TOKEN", "").strip()
    if not token:
        print("refused: HF_TOKEN is not set", file=sys.stderr)
        return 2
    if downloader is None:
        from huggingface_hub import snapshot_download as downloader  # lazy: only this command needs it
    for repo, target in ((args.diarization_repo, args.diarization_dir), (args.embedding_repo, args.embedding_dir)):
        try:
            downloader(repo_id=repo, local_dir=target, token=token, revision=args.revision)
        except Exception:  # noqa: BLE001 - library errors can echo URLs or tokens
            print(f"download of {repo} failed (check the token, the accepted terms and the network)", file=sys.stderr)
            return 1
        print(f"downloaded {repo} to {target}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
