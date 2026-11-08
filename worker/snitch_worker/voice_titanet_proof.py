"""Produce an operator proof for TitaNet using synthetic audio only."""
from __future__ import annotations

import argparse
import json
import sys

from snitch_worker import voice_titanet


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("model_dir")
    parser.add_argument("--device", choices=("auto", "cpu", "cuda"), default="auto")
    parser.add_argument("--release", required=True, help="Manifest of the installed release")
    args = parser.parse_args(argv)
    with open(args.release, encoding="utf-8") as stream:
        release = json.load(stream)
    if release.get("engine") != "nemo-titanet":
        parser.error("release must identify nemo-titanet")
    import torch

    device = args.device
    if device == "auto":
        device = "cpu" if release.get("cuda") == "cpu" else "cuda"
    if (device == "cuda" and not torch.cuda.is_available()) or (
        release.get("cuda") != "cpu" and device == "cpu"
    ):
        print(json.dumps({"ok": False, "refused": "cuda_unavailable"}))
        return 1
    try:
        result = voice_titanet.probe(args.model_dir, device)
    except voice_titanet.TitaNetRefused as refusal:
        print(json.dumps({"ok": False, "refused": refusal.detail}))
        return 1
    result.update(ok=True, release_version=release["version"])
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
