"""Net speech and the quality measures of a fingerprint (ADR 0064, #389). Standard library only.

The net speech of a local speaker is its turns minus the moments another local speaker of the same
track speaks. Measures describe a fingerprint; they never decide its eligibility: Snitch does.
"""

from __future__ import annotations

import array
import math
import sys
import wave
from typing import Callable, Sequence

# Version of the measures carried by a fingerprint result. Raise it whenever a definition below changes.
MEASURES_VERSION = 1
# A digital silence has no level in dBFS; it is reported at this floor so the value stays finite.
LEVEL_FLOOR_DBFS = -120.0
_FULL_SCALE = 32768.0
_CLIPPED = 32767  # a 16-bit sample at (or beyond) full scale, positive or negative

Interval = tuple[float, float]
Levels = Callable[[Sequence[Interval]], "tuple[float, float] | None"]


def union(intervals: Sequence[Interval]) -> list[Interval]:
    merged: list[list[float]] = []
    for start, end in sorted((float(s), float(e)) for s, e in intervals if e > s):
        if merged and start <= merged[-1][1]:
            merged[-1][1] = max(merged[-1][1], end)
        else:
            merged.append([start, end])
    return [(s, e) for s, e in merged]


def subtract(interval: Interval, cuts: Sequence[Interval]) -> list[Interval]:
    """``interval`` minus the (merged, sorted) ``cuts``."""
    start, end = interval
    pieces: list[Interval] = []
    for cut_start, cut_end in cuts:
        if cut_end <= start or cut_start >= end:
            continue
        if cut_start > start:
            pieces.append((start, cut_start))
        start = max(start, cut_end)
        if start >= end:
            break
    if end > start:
        pieces.append((start, end))
    return pieces


def length(intervals: Sequence[Interval]) -> float:
    return sum(e - s for s, e in intervals)


def others_speech(speakers: Sequence[dict], label: str) -> list[Interval]:
    """Where any other local speaker of the track speaks."""
    return union([(float(t["start"]), float(t["end"])) for s in speakers if s["label"] != label for t in s.get("turns", [])])


def overlap_ratio(own: Sequence[Interval], others: Sequence[Interval]) -> float:
    """Share of the speech of ``own`` during which another local speaker speaks, from 0 to 1."""
    speech = union(own)
    total = length(speech)
    if total <= 0:
        return 0.0
    net = sum(length(subtract(piece, others)) for piece in speech)
    return round(min(1.0, max(0.0, (total - net) / total)), 4)


def coherence(vectors: Sequence[Sequence[float]]) -> float | None:
    """Mean pairwise cosine between the turn vectors of one local speaker; absent below two turns."""
    if len(vectors) < 2:
        return None
    units = []
    for vector in vectors:
        norm = math.sqrt(sum(v * v for v in vector)) or 1.0
        units.append([v / norm for v in vector])
    pairs = [sum(a * b for a, b in zip(units[i], units[j])) for i in range(len(units)) for j in range(i + 1, len(units))]
    return round(max(-1.0, min(1.0, sum(pairs) / len(pairs))), 4)


def wav_levels(path: str | None) -> Levels:
    """Clipped-sample share and mean (RMS) level in dBFS of intervals of a prepared 16-bit WAV.

    The prepared audio is the mono 16 kHz track the Worker made with ffmpeg; a file that is absent or
    not a 16-bit WAV gives no audio measures (``None``) rather than failing the fingerprint.
    """

    def measure(intervals: Sequence[Interval]) -> tuple[float, float] | None:
        if not path or not intervals:
            return None
        try:
            with wave.open(path, "rb") as audio:
                if audio.getsampwidth() != 2:
                    return None
                # Every channel of a frame counts: the clipping of one is a clipping of the track.
                rate, frames = audio.getframerate(), audio.getnframes()
                count = clipped = 0
                energy = 0.0
                for start, end in intervals:
                    first = max(0, min(frames, int(start * rate)))
                    last = max(first, min(frames, int(end * rate)))
                    if last == first:
                        continue
                    audio.setpos(first)
                    samples = array.array("h")
                    data = audio.readframes(last - first)
                    samples.frombytes(data[: len(data) - len(data) % 2])
                    if sys.byteorder == "big":
                        samples.byteswap()
                    count += len(samples)
                    clipped += sum(1 for v in samples if v >= _CLIPPED or v <= -_CLIPPED)
                    energy += sum(float(v) * v for v in samples)
        except (wave.Error, EOFError, OSError):
            return None
        if count == 0:
            return None
        rms = math.sqrt(energy / count)
        level = 20 * math.log10(rms / _FULL_SCALE) if rms > 0 else LEVEL_FLOOR_DBFS
        return round(clipped / count, 6), round(max(LEVEL_FLOOR_DBFS, min(0.0, level)), 2)

    return measure
