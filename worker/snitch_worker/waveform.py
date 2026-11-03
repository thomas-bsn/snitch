"""Amplitude peaks for the waveform drawn with a clip (ADR 0034).

The Worker already prepares a mono 16-bit WAV for transcription and diarization;
this reads it once, in bounded chunks, and keeps one normalised maximum per bucket.
The values carry no voice information.
"""

from __future__ import annotations

import array
import sys
import wave
from typing import Any

PEAK_COUNT = 1200
_CHUNK_FRAMES = 1 << 18
_FULL_SCALE = 32768
_LEVELS = 255


def compute_peaks(wav_path: str, count: int = PEAK_COUNT) -> list[int]:
    """Peaks of a prepared WAV, or no peaks when it cannot be read: they are optional."""
    try:
        return _read_peaks(wav_path, count)
    except (wave.Error, EOFError, OSError):
        return []


def _read_peaks(wav_path: str, count: int) -> list[int]:
    with wave.open(wav_path, "rb") as audio:
        if audio.getsampwidth() != 2:
            return []
        total = audio.getnframes()
        channels = audio.getnchannels()
        if total == 0 or count <= 0:
            return []
        bucket = max(1, -(-total // count))
        # Whole buckets per read keep the bucket boundaries exact across chunks.
        per_read = bucket * max(1, _CHUNK_FRAMES // bucket)
        peaks: list[int] = []
        while True:
            data = audio.readframes(per_read)
            if not data:
                break
            samples = array.array("h")
            samples.frombytes(data[: len(data) - len(data) % 2])
            if sys.byteorder == "big":
                samples.byteswap()
            if channels > 1:
                samples = samples[::channels]
            for start in range(0, len(samples), bucket):
                loudest = max(abs(value) for value in samples[start : start + bucket])
                peaks.append(min(_LEVELS, round(min(loudest, _FULL_SCALE) * _LEVELS / _FULL_SCALE)))
    return peaks


def attach_peaks(result: dict[str, Any], wav_path: str) -> None:
    """Add the peaks of the prepared audio to a worker result, in place."""
    result["peaks"] = compute_peaks(wav_path)
