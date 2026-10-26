"""Voice steps (diarization and fingerprints) of the phase 5 worker.

Only the deterministic fake provider exists here. Real providers are opt-in per
pipeline configuration and must never download a model implicitly: until one is
wired, any other provider fails with an explicit, redacted ``provider_unavailable``.
"""

from __future__ import annotations

import hashlib
import math
import re
from typing import Any

DIARIZATION_STEP = re.compile(r"voice\.diarization\.(\d+)")
FINGERPRINT_STEP = re.compile(r"voice\.fingerprint\.(\d+)")
SHORT_TURN_SECONDS = 0.5


class VoiceProviderUnavailable(RuntimeError):
    """Raised for providers that are not installed or not authorised."""


def is_voice_step(step: str) -> bool:
    return bool(DIARIZATION_STEP.fullmatch(step) or FINGERPRINT_STEP.fullmatch(step))


def _fake_settings(config: dict[str, Any]) -> tuple[int, int]:
    if config.get("provider") != "fake":
        raise VoiceProviderUnavailable("voice provider is not available")
    fake = config.get("fake") or {}
    return int(fake.get("speakers", 2)), int(fake.get("dimensions", 8))


def fake_diarization(config: dict[str, Any], duration: float | None, stream_index: int) -> dict[str, Any]:
    """Alternate N speakers over equal slices of the clip, deterministically."""
    speakers, _ = _fake_settings(config)
    total = duration if duration and duration > 0 else 10.0
    slice_length = total / (2 * speakers)
    turns: list[list[dict[str, float]]] = [[] for _ in range(speakers)]
    for position in range(2 * speakers):
        start = position * slice_length
        turns[position % speakers].append({"start": round(start, 6), "end": round(start + slice_length, 6)})
    return {
        "provider": config["provider"],
        "model": config["model"],
        "model_version": config["model_version"],
        "audio_stream_index": stream_index,
        "speakers": [{"label": f"SPEAKER_{i:02d}", "turns": turns[i]} for i in range(speakers)],
    }


def fake_embedding(label: str, dimensions: int) -> list[float]:
    digest = hashlib.sha256(f"fake-voice:{label}".encode()).digest()
    raw = [((digest[i % len(digest)] / 255.0) * 2 - 1) + (i // len(digest)) * 0.01 for i in range(dimensions)]
    norm = math.sqrt(sum(v * v for v in raw)) or 1.0
    return [round(v / norm, 6) for v in raw]


def fake_fingerprints(config: dict[str, Any], diarization: dict[str, Any], stream_index: int) -> dict[str, Any]:
    _, dimensions = _fake_settings(config)
    items: list[dict[str, Any]] = []
    for speaker in diarization.get("speakers", []):
        turns = speaker.get("turns", [])
        if not turns:
            continue
        label = speaker["label"]
        speaking = sum(t["end"] - t["start"] for t in turns)
        first = min(t["start"] for t in turns)
        items.append({
            "speaker_label": label,
            "scope": "speaker",
            "start": first,
            "end": round(first + speaking, 6),
            "quality": 0.9,
            "embedding": fake_embedding(label, dimensions),
        })
        for turn in turns:
            length = turn["end"] - turn["start"]
            items.append({
                "speaker_label": label,
                "scope": "turn",
                "turn_index": turn["index"],
                "start": turn["start"],
                "end": turn["end"],
                "quality": 0.9 if length >= SHORT_TURN_SECONDS else 0.3,
                "embedding": fake_embedding(f"{label}:{turn['index']}", dimensions),
            })
    return {
        "provider": config["provider"],
        "embedding_model": "fake-voice-embedding",
        "embedding_model_version": "fake-v1",
        "audio_stream_index": stream_index,
        "diarization_job_id": diarization["job_id"],
        "dimensions": dimensions,
        "fingerprints": items,
    }


def process_voice_step(step: str, voice: dict[str, Any], duration_seconds: float | None) -> dict[str, Any]:
    """Compute the result of a leased voice step from its pinned input only."""
    config = voice.get("config") or {}
    match = DIARIZATION_STEP.fullmatch(step)
    if match:
        return fake_diarization(config, duration_seconds, int(match.group(1)))
    match = FINGERPRINT_STEP.fullmatch(step)
    if match and voice.get("diarization"):
        return fake_fingerprints(config, voice["diarization"], int(match.group(1)))
    raise ValueError("unsupported voice step")
