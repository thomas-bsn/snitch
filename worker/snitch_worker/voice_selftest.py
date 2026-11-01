"""Synthetic voice inference: proves the installed models run, without any real media.

The installer launcher runs this once after the voice models are in place. Only a
successful run lets Snitch show the voice as ready; downloading models is not enough.
"""
from __future__ import annotations

import math
import os
import struct
import sys
import tempfile
import wave
from collections.abc import Mapping
from typing import Any

from snitch_worker import voice
from snitch_worker.voice import VoiceProviderUnavailable

CONFIG: dict[str, Any] = {
    "provider": "pyannote",
    "model": "pyannote/speaker-diarization-community-1",
    "model_version": "community-1",
    "min_fingerprint_seconds": 1.0,
    "min_quality": 0.4,
}


def write_synthetic_audio(path: str, seconds: float = 3.0, rate: int = 16000) -> None:
    """Deterministic mono tone: no personal data, nothing recorded."""
    with wave.open(path, "wb") as out:
        out.setnchannels(1)
        out.setsampwidth(2)
        out.setframerate(rate)
        frames = (int(9000 * math.sin(2 * math.pi * 220 * i / rate)) for i in range(int(seconds * rate)))
        out.writeframes(b"".join(struct.pack("<h", f) for f in frames))


def run_selftest(providers: Mapping[str, Any] | None = None) -> bool:
    """True when diarization ran end to end on synthetic audio."""
    available = providers or voice.default_providers()
    try:
        voice.ensure_provider_ready({"config": CONFIG}, available)
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "synthetic.wav")
            write_synthetic_audio(path)
            available[CONFIG["provider"]].diarize(CONFIG, path, 0)
    except VoiceProviderUnavailable:
        return False
    return True


def main() -> int:  # pragma: no cover - needs models
    return 0 if run_selftest() else 1


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
