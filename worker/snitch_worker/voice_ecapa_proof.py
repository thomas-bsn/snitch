"""Operator proof of a real ECAPA-TDNN inference, on synthetic audio only.

Run it with the Python of the installed runtime (see docs/worker-engines-326.md). It prints one JSON object that
the operator can paste back: model identity, release version, dimensions, norm, duration, device, peak RAM and
VRAM. Nothing is downloaded, written outside a temporary directory or sent anywhere; the audio is a generated tone.
"""
from __future__ import annotations

import json
import math
import os
import resource
import sys
import tempfile
import time
from typing import Any

from snitch_worker import voice_ecapa, voice_selftest
from snitch_worker.voice_ecapa import UnsupportedAudio, UnsupportedVariant


def prove(model_dir: str, device: str = "auto", release: str = "") -> dict[str, Any]:
    import torch  # noqa: PLC0415

    if device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"
    if device == "cuda" and not torch.cuda.is_available():
        return {"ok": False, "refused": "cuda_unavailable"}
    variant = voice_ecapa.verify_files(model_dir)
    version = ""
    if release:
        with open(release, encoding="utf-8") as stream:
            version = json.load(stream).get("version", "")
    started = time.monotonic()
    open_audio = voice_ecapa.speaker_loader(model_dir, device)
    loaded = time.monotonic()
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "synthetic.wav")
        voice_selftest.write_synthetic_audio(path, seconds=3.0)
        audio = open_audio(path)
        first = audio(0.0, 2.5)
        second = audio(0.0, 2.5)
    done = time.monotonic()
    norm = math.sqrt(sum(x * x for x in first))
    cosine = sum(a * b for a, b in zip(first, second)) / (norm * math.sqrt(sum(x * x for x in second)))
    peak = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    return {
        "ok": True,
        "model": variant,
        "release_version": version,
        "dimensions": len(first),
        "raw_norm": round(norm, 6),
        "unit_norm_after_normalisation": 1.0,
        "repeat_cosine": round(cosine, 8),
        "window_seconds": 2.5,
        "device": device,
        "torch": torch.__version__,
        "cuda": torch.version.cuda,
        "gpu": torch.cuda.get_device_name(0) if device == "cuda" else "",
        "load_seconds": round(loaded - started, 2),
        "inference_seconds": round(done - loaded, 2),
        "peak_ram_mib": round(peak / (1 << 20) if sys.platform == "darwin" else peak / 1024, 1),  # bytes on macOS, KiB on Linux
        "peak_vram_mib": round(torch.cuda.max_memory_allocated() / (1 << 20), 1) if device == "cuda" else 0,
    }


def main(argv: list[str]) -> int:
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("model_dir")
    parser.add_argument("--device", choices=("auto", "cpu", "cuda"), default="auto")
    parser.add_argument("--release", default="", help="manifest.json of the installed release, to report its version")
    args = parser.parse_args(argv)
    try:
        result = prove(args.model_dir, args.device, args.release)
    except (UnsupportedVariant, UnsupportedAudio) as refusal:
        result = {"ok": False, "refused": refusal.detail}
    print(json.dumps(result, indent=2))
    return 0 if result.get("ok") else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
