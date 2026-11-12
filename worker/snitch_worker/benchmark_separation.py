"""Separation step of an end-to-end benchmark chain (#402, spec paroles simultanées section 3).

From the chain's turns of one track, only the Chevauchements involving a Local speaker left without any word on the
track (listed by Snitch under the production attribution rule, #403) are separated, with a margin of context, in one
call to the model. Each separated source is transcribed by the external service and brought back to the time of the
track; it is given the Local speaker whose net-speech fingerprint it resembles the most, only when the gap to the next
one is clear. The result keeps the original transcript as it is and lists the words added in a zone: those of a source
given to a Local speaker without any word in that zone. Nothing here is stored as a fingerprint: the vectors live only for the task.
"""
from __future__ import annotations

import os
import tempfile
import wave
from collections.abc import Callable, Mapping, Sequence
from typing import Any

from snitch_worker import cancellation, runner, voice_separation
from snitch_worker.voice import process_voice_step, turn_interval

# Context kept around a zone, so the model and the transcription hear the start and end of what is said.
MARGIN_SECONDS = 1.0


def _overlap(a: tuple[float, float], b: tuple[float, float]) -> float:
    return max(0.0, min(a[1], b[1]) - max(a[0], b[0]))


def _turns(speaker: Mapping[str, Any]) -> list[tuple[float, float]]:
    return [turn_interval(t) for t in speaker.get("turns") or []]


def _overlaps(speakers: Sequence[Mapping[str, Any]]) -> list[tuple[float, float]]:
    """The merged intervals where at least two Local speakers of the track speak."""
    pieces = sorted((max(a[0], b[0]), min(a[1], b[1]))
                    for i, first in enumerate(speakers) for second in speakers[i + 1:]
                    for a in _turns(first) for b in _turns(second) if _overlap(a, b) > 0)
    merged: list[tuple[float, float]] = []
    for start, end in pieces:
        if merged and start <= merged[-1][1]:
            merged[-1] = (merged[-1][0], max(merged[-1][1], end))
        else:
            merged.append((start, end))
    return merged


def _attribute(vector: Sequence[float] | None, candidates: Mapping[str, Sequence[float]], margin: float) -> str | None:
    """The closest Local speaker by cosine (unit vectors), or None without a clear gap to the next one."""
    if vector is None or not candidates:
        return None
    scores = sorted(((sum(a * b for a, b in zip(vector, v)), label) for label, v in candidates.items()), reverse=True)
    if len(scores) > 1 and scores[0][0] - scores[1][0] < margin:
        return None
    return scores[0][1]


def _speaker_vectors(fingerprints: Mapping[str, Any]) -> dict[str, list[float]]:
    return {f["speaker_label"]: f["embedding"] for f in fingerprints.get("fingerprints") or [] if f.get("scope") == "speaker"}


def run(voice: dict[str, Any], separation: Mapping[str, Any], wav_path: str, stream: int, transcriber: Any,
        pipeline: Mapping[str, Any], providers: Mapping[str, Any] | None, environ: Mapping[str, str],
        separate: Callable[..., list[dict[str, Any]]] = voice_separation.separate) -> dict[str, Any]:
    """Run the separation step on the prepared track ``wav_path``; every time is in the time of the track."""
    speakers = voice["diarization"]["speakers"]
    # Snitch decides who is without words with the production attribution rule (#403): the Worker has no rule of its own.
    silent = list(separation["silent_speakers"])
    zones = []
    for start, end in _overlaps(speakers):
        present = [s["label"] for s in speakers if any(_overlap((start, end), t) > 0 for t in _turns(s))]
        if any(label in silent for label in present):
            zones.append({"start": start, "end": end, "silent_speakers": [label for label in present if label in silent]})
    result: dict[str, Any] = {"audio_stream_index": stream, "silent_speakers": silent, "zones": zones, "sources": [], "added": []}
    if not zones:
        return result
    # shortcut: a zone longer than the model's 60 s window fails the call; clips last 15-30 s, split it if that changes.
    windows = [(max(0.0, z["start"] - MARGIN_SECONDS), z["end"] + MARGIN_SECONDS) for z in zones]
    fingerprint_step = f"voice.fingerprint.{stream}"
    with tempfile.TemporaryDirectory(prefix="snitch-separation-") as temporary:
        separated = separate(environ, separation["model"], separation["revision"], wav_path, temporary, windows)
        local = _speaker_vectors(process_voice_step(fingerprint_step, voice, None, wav_path, providers))
        for number, (zone, window) in enumerate(zip(zones, separated)):
            for source in window["sources"]:
                cancellation.check()
                prepared = os.path.join(temporary, f"prepared-{number}-{source['index']}.wav")
                runner.prepare_audio(source["path"], prepared, 0)  # the rate of the model to the 16 kHz of the pipeline
                with wave.open(prepared, "rb") as audio:
                    seconds = audio.getnframes() / audio.getframerate()
                own = {"job_id": voice["diarization"]["job_id"], "speakers": [{"label": "source", "turns": [{"index": 0, "start": 0.0, "end": seconds}]}]}
                vector = _speaker_vectors(process_voice_step(fingerprint_step, {**voice, "diarization": own}, None, prepared, providers)).get("source")
                speaker = _attribute(vector, local, float(separation["ambiguity_margin"]))
                transcript = runner.transcribe_prepared_audio(transcriber, prepared, dict(pipeline), [], [(0.0, seconds)], float(window["start"]))
                said = [{"start": round(w["start"], 6), "end": round(w["end"], 6), "word": w["word"]}
                        for s in transcript.get("segments") or [] for w in s.get("words") or []]
                result["sources"].append({"zone": number, "index": source["index"], "speaker": speaker, "words": said})
                # Merged zone by zone (spec section 3): a Local speaker with words elsewhere on the track but none here
                # gets them too; its word times come from Snitch's rule, like the speakers without words.
                if speaker is not None and not any(_overlap((zone["start"], zone["end"]), (s["start"], s["end"])) > 0
                                                   for s in separation["word_spans"] if s["speaker"] == speaker):
                    result["added"].extend({"zone": number, "source": source["index"], "speaker": speaker, **w}
                                           for w in said if _overlap((zone["start"], zone["end"]), (w["start"], w["end"])) > 0)
    return result
