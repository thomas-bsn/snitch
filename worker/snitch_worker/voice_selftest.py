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


def diarize_synthetic(provider: Any, provider_name: str = "pyannote") -> None:
    """Diarize a synthetic tone with ``provider``: raises VoiceProviderUnavailable when the model cannot run."""
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "synthetic.wav")
        write_synthetic_audio(path)
        provider.diarize({**CONFIG, "provider": provider_name}, path, 0)


def check_embedding(vector: Any) -> None:
    """A fingerprint is a non-empty vector of finite numbers with a norm above zero; anything else is a failed proof."""
    try:
        numbers = [float(x) for x in vector]
    except (TypeError, ValueError):
        raise VoiceProviderUnavailable("inference_failed") from None
    if not numbers or not all(math.isfinite(x) for x in numbers) or math.sqrt(sum(x * x for x in numbers)) <= 0:
        raise VoiceProviderUnavailable("inference_failed")


def embed_synthetic(provider: Any) -> None:
    """Load the embedding model of ``provider`` and embed a synthetic tone: never a diarization."""
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "synthetic.wav")
        write_synthetic_audio(path)
        try:
            audio = provider._audio_opener()(path)
            seconds = audio.num_samples / audio.sample_rate
            vector = audio(0.0, min(2.0, seconds))
            end = (audio.num_samples - 1) / audio.sample_rate
            other = audio(seconds / 3, end)
        except VoiceProviderUnavailable:
            raise
        except Exception:  # noqa: BLE001 - a library message never travels
            raise VoiceProviderUnavailable("inference_failed") from None
    check_embedding(vector)
    check_embedding(other)
    # The dimension of a fingerprint is the model's own: it must not depend on the audio window.
    if len(list(vector)) != len(list(other)):
        raise VoiceProviderUnavailable("inference_failed")


def write_synthetic_mixture(path: str, seconds: float = 2.0, rate: int = 16000, channels: int = 1) -> None:
    """Deterministic mixture of two synthetic voices (harmonic tones of different pitch, each with its own slow
    amplitude swell): no personal data, nothing recorded. It only proves that a separation model runs."""
    def voice(i: int, pitch: float, swell: float) -> float:
        t = i / rate
        envelope = 0.55 + 0.45 * math.sin(2 * math.pi * swell * t)
        return envelope * sum(math.sin(2 * math.pi * pitch * harmonic * t) / harmonic for harmonic in (1, 2, 3))

    with wave.open(path, "wb") as out:
        out.setnchannels(channels)
        out.setsampwidth(2)
        out.setframerate(rate)
        frames = (int(4500 * (voice(i, 140.0, 1.5) + voice(i, 233.0, 2.5))) for i in range(int(seconds * rate)))
        out.writeframes(b"".join(struct.pack("<h", f) * channels for f in frames))


def separate_synthetic(separate: Any, sources: int, sample_rate: int) -> None:
    """Test inference of a separation model: ``separate(audio_path, output_dir)`` must turn a synthetic mixture
    into the ``sources`` voices of the model, each a WAV at ``sample_rate`` as long as the mixture, and at least
    one of them must carry signal. Anything else is a failed proof; nothing is kept."""
    seconds = 2.0
    with tempfile.TemporaryDirectory() as tmp:
        path = os.path.join(tmp, "mixture.wav")
        write_synthetic_mixture(path, seconds, sample_rate)
        output = os.path.join(tmp, "sources")
        os.makedirs(output)
        try:
            result = separate(path, output)
        except VoiceProviderUnavailable:
            raise
        except Exception:  # noqa: BLE001 - a library message never travels
            raise VoiceProviderUnavailable("inference_failed") from None
        try:
            written = result["sources"]
            if not isinstance(written, list) or len(written) != sources or result["sample_rate"] != sample_rate:
                raise ValueError()
            loudest = 0
            for source in written:
                with wave.open(source["path"], "rb") as stored:
                    if (stored.getframerate(), stored.getnchannels(), stored.getsampwidth(), stored.getnframes()) != (sample_rate, 1, 2, int(seconds * sample_rate)):
                        raise ValueError()
                    frames = stored.readframes(stored.getnframes())
                loudest = max(loudest, max(abs(sample) for (sample,) in struct.iter_unpack("<h", frames)))
            if loudest == 0:
                raise ValueError()
        except (KeyError, TypeError, ValueError, OSError, EOFError, wave.Error, struct.error):
            raise VoiceProviderUnavailable("inference_failed") from None


def run_selftest(providers: Mapping[str, Any] | None = None, report: list[str] | None = None) -> bool:
    """True when diarization ran end to end on synthetic audio; ``report`` receives the stable failure code."""
    available = providers or voice.default_providers()
    try:
        voice.ensure_provider_ready({"config": CONFIG}, available, step="voice.diarization.0")
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "synthetic.wav")
            write_synthetic_audio(path)
            available[CONFIG["provider"]].diarize(CONFIG, path, 0)
    except VoiceProviderUnavailable as error:
        if report is not None:
            report.append(str(getattr(error, "reason", "provider_unavailable")))
        return False
    return True


def main() -> int:  # pragma: no cover - needs models
    report: list[str] = []
    if run_selftest(report=report):
        return 0
    # Stable machine code only (never a path, token or library message): the launcher shows it.
    print("selftest_reason=" + (report[0] if report else "provider_unavailable"), file=sys.stderr)
    return 1


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
