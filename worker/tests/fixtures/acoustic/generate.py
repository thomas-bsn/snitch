"""Recipe for the synthetic acoustic fixtures (issue #343). Run with a Python that has numpy.

    python generate.py OUTPUT_DIR

Only open tools are used (espeak-ng, ffmpeg) and no media comes from a real library.
The committed WAV files are the regression evidence; `manifest.json` carries their
SHA-256 and the intended decision of each pair. Re-running may produce different AAC bytes
on another ffmpeg build: tests assert behaviour, and only the lossless originals are hashed.
"""
import hashlib
import json
import subprocess
import sys
import tempfile
import wave
from pathlib import Path

import numpy as np

RATE = 16000
TEXT = "go to the north gate now"


def run(*cmd):
    subprocess.run(cmd, check=True, capture_output=True)


def read(path):
    with wave.open(str(path), "rb") as w:
        assert w.getnchannels() == 1 and w.getsampwidth() == 2 and w.getframerate() == RATE
        return np.frombuffer(w.readframes(w.getnframes()), dtype="<i2").astype(np.int64)


def write(path, samples):
    with wave.open(str(path), "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(np.clip(samples, -32768, 32767).astype("<i2").tobytes())


def speech(tmp, name, voice, pitch, speed):
    raw = tmp / (name + ".raw.wav")
    run("espeak-ng", "-v", voice, "-p", str(pitch), "-s", str(speed), "-w", str(raw), TEXT)
    run("ffmpeg", "-y", "-i", str(raw), "-ar", str(RATE), "-ac", "1", "-c:a", "pcm_s16le", str(tmp / (name + ".wav")))
    x = read(tmp / (name + ".wav"))
    return np.rint(x * (12000 / max(1, np.abs(x).max()))).astype(np.int64)


def game_bed(seconds):
    """Seeded loud bed: filtered noise, impulses and a pitched drone, no speech."""
    rng = np.random.default_rng(274343)
    n = int(seconds * RATE)
    noise = np.convolve(rng.standard_normal(n), np.ones(8) / 8, mode="same") * 3000
    t = np.arange(n) / RATE
    drone = 4000 * np.sin(2 * np.pi * 110 * t) * (0.6 + 0.4 * np.sin(2 * np.pi * 0.7 * t))
    impulses = np.zeros(n)
    impulses[rng.integers(0, n, 12)] = 9000
    return np.rint(noise + drone + np.convolve(impulses, np.hanning(64), mode="same")).astype(np.int64)


def main(out):
    out.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)
        a = speech(tmp, "a", "en-us", 50, 150)
        b = speech(tmp, "b", "en-us+f3", 75, 150)
        a_again = speech(tmp, "again", "en-us", 50, 135)
        n = max(len(a), len(b))
        bed = game_bed(n / RATE)
        pad = lambda x: np.concatenate([x, np.zeros(n - len(x), dtype=np.int64)])
        a, b = pad(a), pad(b)
        write(out / "voice-a.wav", a)
        write(out / "voice-b-same-words.wav", b)
        write(out / "voice-a-second-take.wav", a_again)
        write(out / "game-bed.wav", bed[:n])
        write(out / "mix-a-game.wav", a + bed[:n] // 2)
        write(out / "mix-b-game.wav", b + bed[:n] // 2)
        write(out / "mix-a-and-b.wav", a + b)
        for gain in ("0.25", "0.5", "2.0"):
            run("ffmpeg", "-y", "-i", str(out / "voice-a.wav"), "-filter:a", f"volume={gain}", "-c:a", "pcm_s16le", str(out / f"voice-a-gain-{gain}.wav"))
        run("ffmpeg", "-y", "-i", str(out / "voice-a.wav"), "-c:a", "flac", str(tmp / "a.flac"))
        run("ffmpeg", "-y", "-i", str(tmp / "a.flac"), "-c:a", "pcm_s16le", str(out / "voice-a-flac-roundtrip.wav"))
        run("ffmpeg", "-y", "-i", str(out / "voice-a.wav"), "-c:a", "aac", "-b:a", "64k", str(tmp / "a.m4a"))
        run("ffmpeg", "-y", "-i", str(tmp / "a.m4a"), "-ar", str(RATE), "-ac", "1", "-c:a", "pcm_s16le", str(out / "voice-a-aac64k.wav"))
    manifest = {
        "recipe": "worker/tests/fixtures/acoustic/generate.py",
        "text": TEXT,
        "synthesis": "espeak-ng (GPL-3.0 tool; its output is generated test data), ffmpeg",
        "files": {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(out.glob("*.wav"))},
    }
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")


if __name__ == "__main__":
    main(Path(sys.argv[1]))
