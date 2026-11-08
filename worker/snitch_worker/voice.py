"""Voice steps (diarization and fingerprints) of the phase 5 worker.

The deterministic fake provider and the real ``pyannote`` provider
(``voice_pyannote``) are selected by the pipeline configuration. Real providers
never download a model implicitly; any provider that is unknown, not installed or
not authorised fails with an explicit, redacted ``provider_unavailable``.
"""

from __future__ import annotations

import hashlib
import logging
import math
import re
from typing import Any, Callable, Mapping, Sequence

from snitch_worker import cancellation

logger = logging.getLogger(__name__)

DIARIZATION_STEP = re.compile(r"voice\.diarization\.(\d+)")
FINGERPRINT_STEP = re.compile(r"voice\.fingerprint\.(\d+)")
SHORT_TURN_SECONDS = 0.5
# Limits enforced by the API on a voice result; exceeding them would be rejected
# after the (costly) computation, so real providers fail early instead.
MAX_SPEAKERS = 32
MAX_TURNS = 2000
MAX_FINGERPRINTS = 400
# Turns shorter than this are not embedded at all (speaker models need a minimum window).
MIN_EMBEDDABLE_SECONDS = 0.5
# Heuristic, not calibrated: a turn of this length (a speaker of ten times this
# length) is considered fully usable; shorter ones get a proportional quality.
FULL_QUALITY_TURN_SECONDS = 3.0
FULL_QUALITY_SPEAKER_SECONDS = 10.0

REAL_PROVIDERS = frozenset({"pyannote", "nemo"})


class VoiceProviderUnavailable(RuntimeError):
    """Raised for providers that are not installed or not authorised.

    The message is constant on purpose: it never carries a path, a token or a
    library error. ``reason`` is a stable machine code that is safe to log.
    """

    def __init__(self, reason: str = "provider_unavailable") -> None:
        super().__init__("voice provider is not available")
        self.reason = reason


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


def requires_audio(config: dict[str, Any]) -> bool:
    """Real providers read the prepared audio; the fake one never touches media."""
    return config.get("provider") in REAL_PROVIDERS


def _clean_turns(turns: Sequence[tuple[float, float, str]]) -> list[tuple[float, float, str]]:
    return [(float(s), float(e), label) for s, e, label in turns if float(e) > float(s) >= 0.0]


def diarization_from_turns(config: dict[str, Any], turns: Sequence[tuple[float, float, str]], stream_index: int) -> dict[str, Any]:
    """Build the diarization result from raw ``(start, end, label)`` turns.

    Speakers are renamed ``SPEAKER_00``... in order of first appearance, turns
    are sorted, empty turns dropped.
    """
    ordered = sorted(_clean_turns(turns), key=lambda t: (t[0], t[1]))
    names: dict[str, str] = {}
    grouped: dict[str, list[dict[str, float]]] = {}
    for start, end, raw in ordered:
        label = names.setdefault(raw, f"SPEAKER_{len(names):02d}")
        grouped.setdefault(label, []).append({"start": round(start, 6), "end": round(end, 6)})
    if len(grouped) > MAX_SPEAKERS or len(ordered) > MAX_TURNS:
        raise VoiceProviderUnavailable("result_too_large")
    return {
        "provider": config["provider"],
        "model": config["model"],
        "model_version": config["model_version"],
        "audio_stream_index": stream_index,
        "speakers": [{"label": label, "turns": grouped[label]} for label in grouped],
    }


def _unit(vector: Sequence[float]) -> list[float]:
    values = [float(v) for v in vector]
    if not values or any(not math.isfinite(v) for v in values):
        raise VoiceProviderUnavailable("invalid_embedding")
    norm = math.sqrt(sum(v * v for v in values))
    if norm == 0.0:
        raise VoiceProviderUnavailable("invalid_embedding")
    return [round(v / norm, 6) for v in values]


def fingerprints_from_embedder(
    config: dict[str, Any],
    diarization: dict[str, Any],
    embed: Callable[[float, float], Sequence[float] | None],
    stream_index: int,
    model: str,
    model_version: str,
) -> dict[str, Any]:
    """One unit vector per speaker (duration-weighted mean) and per embeddable turn."""
    per_speaker: list[tuple[str, list[dict[str, Any]], list[float], float]] = []
    dimensions = 0
    for speaker in diarization.get("speakers", []):
        turns: list[dict[str, Any]] = []
        for turn in speaker.get("turns", []):
            length = turn["end"] - turn["start"]
            if length < MIN_EMBEDDABLE_SECONDS:
                continue
            cancellation.check()  # between two embeddings, whatever the provider
            raw = embed(float(turn["start"]), float(turn["end"]))
            if raw is None:  # no audio left under this turn: kept in the diarization, not fingerprinted
                continue
            vector = _unit(raw)
            if dimensions and len(vector) != dimensions:
                raise VoiceProviderUnavailable("invalid_embedding")
            dimensions = len(vector)
            turns.append({"index": turn["index"], "start": turn["start"], "end": turn["end"], "length": length, "embedding": vector})
        if not turns:
            continue
        total = sum(t["length"] for t in turns)
        mean = [sum(t["embedding"][i] * t["length"] for t in turns) / total for i in range(dimensions)]
        per_speaker.append((speaker["label"], turns, _unit(mean), total))
    # Keep the longest turns when the API limit would be exceeded.
    budget = max(0, MAX_FINGERPRINTS - len(per_speaker))
    all_turns = [t for _, turns, _, _ in per_speaker for t in turns]
    if len(all_turns) > budget:
        keep = {id(t) for t in sorted(all_turns, key=lambda t: -t["length"])[:budget]}
    else:
        keep = {id(t) for t in all_turns}
    items: list[dict[str, Any]] = []
    for label, turns, mean_vector, total in per_speaker:
        first = min(t["start"] for t in turns)
        items.append({
            "speaker_label": label,
            "scope": "speaker",
            "start": first,
            "end": round(first + total, 6),
            "quality": round(min(1.0, total / FULL_QUALITY_SPEAKER_SECONDS), 4),
            "embedding": mean_vector,
        })
        for t in turns:
            if id(t) in keep:
                items.append({
                    "speaker_label": label,
                    "scope": "turn",
                    "turn_index": t["index"],
                    "start": t["start"],
                    "end": t["end"],
                    "quality": round(min(1.0, t["length"] / FULL_QUALITY_TURN_SECONDS), 4),
                    "embedding": t["embedding"],
                })
    return {
        "provider": config["provider"],
        "embedding_model": model,
        "embedding_model_version": model_version,
        "audio_stream_index": stream_index,
        "diarization_job_id": diarization["job_id"],
        "dimensions": dimensions or 1,
        "fingerprints": items,
    }


class _FakeProvider:
    def ensure_ready(self, config: dict[str, Any]) -> None:
        _fake_settings(config)

    def diarize(self, config: dict[str, Any], audio_path: str | None, stream_index: int, duration: float | None = None) -> dict[str, Any]:
        return fake_diarization(config, duration, stream_index)

    def fingerprint(self, config: dict[str, Any], diarization: dict[str, Any], audio_path: str | None, stream_index: int) -> dict[str, Any]:
        return fake_fingerprints(config, diarization, stream_index)


_default_providers: dict[str, Any] | None = None


def default_providers() -> dict[str, Any]:
    """Process-wide providers; the pyannote one caches its models between tasks."""
    global _default_providers
    if _default_providers is None:
        from snitch_worker.voice_nemo import NemoProvider  # light: no ML import
        from snitch_worker.voice_pyannote import PyannoteProvider

        from snitch_worker.voice_runtime import RuntimeProvider
        from snitch_worker.engine_bridge import SpeakerProvider
        _default_providers = {"fake": _FakeProvider(), "pyannote": RuntimeProvider(PyannoteProvider()), "nemo": RuntimeProvider(NemoProvider()),
                              "nemo-titanet": RuntimeProvider(SpeakerProvider("nemo-titanet")),
                              "speechbrain-ecapa": RuntimeProvider(SpeakerProvider("speechbrain-ecapa"))}
    return _default_providers


def ensure_provider_ready(voice: dict[str, Any], providers: Mapping[str, Any] | None = None, *, step: str | None = None) -> None:
    """Cheap pre-flight (no model load, no media): fail before downloading anything."""
    config = {**(voice.get("config") or {})}
    if voice.get("fingerprint_model") is not None:
        config["fingerprint_model"] = voice["fingerprint_model"]
    name = config.get("provider")
    if step and FINGERPRINT_STEP.fullmatch(step) and config.get("fingerprint_model") and name != "fake":
        name = config["fingerprint_model"].get("engine", name)
    provider = (providers or default_providers()).get(name)
    if provider is None:
        raise VoiceProviderUnavailable("unknown_provider")
    check = "ensure_ready"
    if step and DIARIZATION_STEP.fullmatch(step):
        check = "ensure_diarization_ready"
    elif step and FINGERPRINT_STEP.fullmatch(step):
        check = "ensure_fingerprint_ready"
    getattr(provider, check, provider.ensure_ready)(config)


def process_voice_step(
    step: str,
    voice: dict[str, Any],
    duration_seconds: float | None,
    audio_path: str | None = None,
    providers: Mapping[str, Any] | None = None,
) -> dict[str, Any]:
    """Compute the result of a leased voice step from its pinned input only."""
    config = {**(voice.get("config") or {})}
    if voice.get("fingerprint_model") is not None:
        config["fingerprint_model"] = voice["fingerprint_model"]
    diarization_match = DIARIZATION_STEP.fullmatch(step)
    fingerprint_match = FINGERPRINT_STEP.fullmatch(step)
    if not diarization_match and not (fingerprint_match and voice.get("diarization")):
        raise ValueError("unsupported voice step")
    cancellation.check()
    ensure_provider_ready(voice, providers, step=step)
    name = config["provider"]
    if fingerprint_match and config.get("fingerprint_model") and name != "fake":
        name = config["fingerprint_model"].get("engine", name)
    provider = (providers or default_providers())[name]
    if requires_audio(config) and not audio_path:
        raise VoiceProviderUnavailable("audio_missing")
    cancellation.check()  # last point before the provider's opaque call
    try:
        if diarization_match:
            stream = int(diarization_match.group(1))
            if isinstance(provider, _FakeProvider):
                return provider.diarize(config, audio_path, stream, duration_seconds)
            return provider.diarize(config, audio_path, stream)
        return provider.fingerprint(config, voice["diarization"], audio_path, int(fingerprint_match.group(1)))
    except VoiceProviderUnavailable as exc:
        logger.warning("voice step failed: %s", exc.reason)
        raise
