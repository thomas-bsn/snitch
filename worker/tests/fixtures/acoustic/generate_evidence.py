"""Regenerates the cross-language evidence sample read by Go (audioplans) from the real adapter.

    PYTHONPATH=worker python worker/tests/fixtures/acoustic/generate_evidence.py > app/internal/audioplans/fixtures/acoustic-evidence.json

Tracks (real stream indices, deliberately not 0..n): 2 original, 7 lossless re-encode, 9 same
words by another voice, 11 gain 0.5, 12 AAC 64k, 13 original plus synthetic game bed.
"""
import json
import sys
import wave
from pathlib import Path

from snitch_worker import acoustic_dedup

HERE = Path(__file__).resolve().parent
TRACKS = {2: "voice-a.wav", 7: "voice-a-flac-roundtrip.wav", 9: "voice-b-same-words.wav", 11: "voice-a-gain-0.5.wav", 12: "voice-a-aac64k.wav", 13: "mix-a-game.wav"}


def events():
    out = []
    for track, name in TRACKS.items():
        with wave.open(str(HERE / name), "rb") as handle:
            seconds = handle.getnframes() / handle.getframerate()
        out.append({"id": f"t{track}", "track_index": track, "job_id": track * 10, "path": str(HERE / name), "clip_offset": 0.0, "start": 0.0, "end": round(seconds, 6)})
    return out


if __name__ == "__main__":
    report = acoustic_dedup.run({"schema_version": 1, "events": events()})
    del report["stats"]  # timings are not reproducible
    json.dump(report, sys.stdout, indent=1, sort_keys=True)
    print()
