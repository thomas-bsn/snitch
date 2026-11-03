"""Integration tests with the real WeSpeaker embedder (pyannote/wespeaker-voxceleb-resnet34-LM).

Opt-in: they need the ``voice`` extra and a local snapshot of the model, given by path so that nothing is
downloaded or read from the library by default::

    SNITCH_WESPEAKER_TEST_PATH=/path/to/snapshot \\
        uv run --frozen python -m unittest tests.test_voice_wespeaker

Audio is synthetic (generated WAV files in a temporary directory): no clip of the real library is used.
The window bounds are checked against pyannote's own crop, which is the constraint they exist for.
"""

import importlib.util
import math
import os
import random
import tempfile
import unittest
import wave

from snitch_worker import voice_pyannote

MODEL_PATH = os.environ.get("SNITCH_WESPEAKER_TEST_PATH", "")
HAVE_STACK = bool(MODEL_PATH) and all(
    importlib.util.find_spec(name) for name in ("torch", "torchaudio", "pyannote.audio")
)
SKIP_REASON = "set SNITCH_WESPEAKER_TEST_PATH to a local WeSpeaker snapshot and install the voice extra"

CONFIG = {
    "provider": "pyannote",
    "model": "pyannote/speaker-diarization-community-1",
    "model_version": "community-1",
    "min_fingerprint_seconds": 1.0,
    "min_quality": 0.4,
}


def write_wav(path, seconds, rate=16000, channels=1):
    """Low-level noise: the embedder runs on it, and the file is a real WAV decoded by torchaudio."""
    rng = random.Random(40)
    frames = round(seconds * rate)
    with wave.open(path, "wb") as out:
        out.setnchannels(channels)
        out.setsampwidth(2)
        out.setframerate(rate)
        out.writeframes(
            b"".join(
                int(rng.uniform(-3000, 3000)).to_bytes(2, "little", signed=True)
                * channels
                for _ in range(frames)
            )
        )
    return frames


@unittest.skipUnless(HAVE_STACK, SKIP_REASON)
class RealWeSpeakerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls._tmp = tempfile.TemporaryDirectory()
        cls.dir = cls._tmp.name
        # 16 kHz mono, 10 s: the reference clip. Every case below is a turn on this file, or on a 44.1 kHz stereo one.
        cls.frames = write_wav(os.path.join(cls.dir, "mono.wav"), 10.0)
        write_wav(os.path.join(cls.dir, "stereo.wav"), 10.0, rate=44100, channels=2)
        cls.stereo_frames = round(10.0 * 44100)
        env = {
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": MODEL_PATH,
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": MODEL_PATH,
            "SNITCH_VOICE_EMBEDDING_MODEL_NAME": "pyannote/wespeaker-voxceleb-resnet34-LM",
            "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "wespeaker-integration",
        }
        cls.provider = voice_pyannote.PyannoteProvider(
            environ=env, device_probe=lambda: "cpu"
        )

    @classmethod
    def tearDownClass(cls):
        cls._tmp.cleanup()

    def fingerprint(self, filename, turns):
        diarization = {
            "job_id": 1,
            "speakers": [
                {
                    "label": "SPEAKER_00",
                    "turns": [
                        {"index": i, "start": s, "end": e}
                        for i, (s, e) in enumerate(turns)
                    ],
                }
            ],
        }
        return self.provider.fingerprint(
            CONFIG, diarization, os.path.join(self.dir, filename), 0
        )

    def turn_indexes(self, result):
        return sorted(
            f["turn_index"] for f in result["fingerprints"] if f["scope"] == "turn"
        )

    def test_the_library_refuses_a_window_ending_on_the_last_sample(self):
        # The constraint behind the bound: pyannote's crop rejects end == duration, accepts the sample before.
        import torch
        import torchaudio
        from pyannote.audio import Inference, Model
        from pyannote.core import Segment

        inference = Inference(
            Model.from_pretrained(MODEL_PATH),
            window="whole",
            device=torch.device("cpu"),
        )
        waveform, rate = torchaudio.load(os.path.join(self.dir, "mono.wav"))
        audio = {"waveform": waveform, "sample_rate": rate}
        last = (self.frames - 1) / rate
        inference.crop(audio, Segment(0.0, last))  # the last valid sample: accepted
        with self.assertRaises(ValueError):
            inference.crop(
                audio, Segment(0.0, self.frames / rate)
            )  # the duration itself: refused

    def test_a_turn_ending_exactly_at_the_end_of_the_wav_is_embedded(self):
        result = self.fingerprint("mono.wav", [(7.0, self.frames / 16000)])
        self.assertEqual(self.turn_indexes(result), [0])

    def test_a_turn_overshooting_the_wav_by_20_ms_is_embedded_and_keeps_its_bounds(
        self,
    ):
        result = self.fingerprint("mono.wav", [(7.0, self.frames / 16000 + 0.02)])
        self.assertEqual(self.turn_indexes(result), [0])
        turn = next(f for f in result["fingerprints"] if f["scope"] == "turn")
        self.assertAlmostEqual(turn["end"], self.frames / 16000 + 0.02, places=6)

    def test_a_turn_ending_one_sample_past_the_wav_is_embedded(self):
        result = self.fingerprint("mono.wav", [(7.0, self.frames / 16000 + 1 / 16000)])
        self.assertEqual(self.turn_indexes(result), [0])

    def test_a_turn_ending_at_the_end_of_a_44_1_khz_stereo_wav_is_embedded(self):
        # Decoded at 44.1 kHz and downmixed by pyannote: the bound must follow the decoded sample count.
        result = self.fingerprint("stereo.wav", [(7.0, self.stereo_frames / 44100)])
        self.assertEqual(self.turn_indexes(result), [0])

    def test_a_turn_too_short_once_cut_at_the_end_is_skipped_without_failing(self):
        # 0.6 s turn, of which only ~0.1 s is left in the audio: not embeddable, the clip still succeeds.
        result = self.fingerprint("mono.wav", [(2.0, 4.0), (9.9, 10.5)])
        self.assertEqual(self.turn_indexes(result), [0])
        self.assertTrue(any(f["scope"] == "speaker" for f in result["fingerprints"]))

    def test_a_turn_of_exactly_the_minimum_length_in_the_middle_is_embedded(self):
        result = self.fingerprint("mono.wav", [(3.0, 3.5)])
        self.assertEqual(self.turn_indexes(result), [0])

    def test_a_turn_of_exactly_the_minimum_length_ending_on_the_last_sample_is_embedded(
        self,
    ):
        last = (self.frames - 1) / 16000
        result = self.fingerprint("mono.wav", [(last - 0.5, self.frames / 16000)])
        self.assertEqual(self.turn_indexes(result), [0])

    def test_turns_are_embedded_as_unit_vectors_of_one_dimension(self):
        result = self.fingerprint(
            "mono.wav", [(1.0, 3.0), (7.0, self.frames / 16000 + 0.02)]
        )
        vectors = [
            f["embedding"] for f in result["fingerprints"] if f["scope"] == "turn"
        ]
        self.assertEqual(len(vectors), 2)
        self.assertEqual(len({len(v) for v in vectors}), 1)
        for vector in vectors:
            self.assertTrue(math.isclose(sum(x * x for x in vector), 1.0, rel_tol=1e-4))


if __name__ == "__main__":
    unittest.main()
