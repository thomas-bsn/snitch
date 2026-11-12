import math
import os
import struct
import tempfile
import unittest
import wave

from snitch_worker import voice_selftest
from snitch_worker.voice import VoiceProviderUnavailable


class Provider:
    def __init__(self, ready=True, fails=False):
        self.ready, self.fails, self.audio = ready, fails, None

    def ensure_ready(self, config):
        if not self.ready:
            raise VoiceProviderUnavailable("model_path_missing")

    def diarize(self, config, audio_path, stream_index):
        if self.fails:
            raise VoiceProviderUnavailable("inference_failed")
        with open(audio_path, "rb") as handle:
            self.audio = handle.read(4)
        return {}


class VoiceSelftestTests(unittest.TestCase):
    def test_success_needs_an_inference_on_synthetic_audio(self):
        provider = Provider()
        self.assertTrue(voice_selftest.run_selftest(providers={"pyannote": provider}))
        self.assertEqual(provider.audio, b"RIFF")

    def test_missing_models_are_not_ready(self):
        self.assertFalse(voice_selftest.run_selftest(providers={"pyannote": Provider(ready=False)}))

    def test_failure_reports_the_stable_reason_code(self):
        report = []
        self.assertFalse(voice_selftest.run_selftest(providers={"pyannote": Provider(ready=False)}, report=report))
        self.assertEqual(len(report), 1)
        self.assertRegex(report[0], r"^[a-z_]+$")

    def test_failed_inference_is_not_ready(self):
        self.assertFalse(voice_selftest.run_selftest(providers={"pyannote": Provider(fails=True)}))


if __name__ == "__main__":
    unittest.main()


class EmbeddingProofTests(unittest.TestCase):
    def test_a_valid_embedding_passes(self):
        voice_selftest.check_embedding([0.1, -0.2, 0.3])

    def test_a_bad_embedding_fails_with_a_stable_reason(self):
        for bad in ([], [0.0, 0.0], [float("nan"), 1.0], [float("inf")], None, ["x"]):
            with self.assertRaises(VoiceProviderUnavailable) as caught:
                voice_selftest.check_embedding(bad)
            self.assertEqual(caught.exception.reason, "inference_failed")

    def test_the_synthetic_embedding_runs_the_embedder_and_not_the_diarization(self):
        from snitch_worker.voice_pyannote import DecodedAudio, PyannoteProvider

        seen = []

        def loader(path, device):
            def open_audio(audio_path):
                seen.append(audio_path)
                return DecodedAudio(lambda start, end: [0.5, 0.5], 16000, 1, 48000)

            return open_audio

        provider = PyannoteProvider(environ={"SNITCH_VOICE_EMBEDDING_MODEL_PATH": "/", "SNITCH_VOICE_DEVICE": "cpu"}, load_embedder=loader)
        voice_selftest.embed_synthetic(provider)
        self.assertEqual(len(seen), 1)

    def test_an_embedding_whose_dimension_changes_with_the_window_fails(self):
        from snitch_worker.voice_pyannote import DecodedAudio, PyannoteProvider

        def loader(path, device):
            return lambda audio_path: DecodedAudio(lambda start, end: [0.5] * (2 if start == 0.0 else 3), 16000, 1, 48000)

        provider = PyannoteProvider(environ={"SNITCH_VOICE_EMBEDDING_MODEL_PATH": "/", "SNITCH_VOICE_DEVICE": "cpu"}, load_embedder=loader)
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_selftest.embed_synthetic(provider)
        self.assertEqual(caught.exception.reason, "inference_failed")


def stored_sources(amplitudes, rate=16000, samples=None, sources=None):
    """A fake separation: it reads the mixture it was given and writes one source per amplitude."""
    seen = []

    def separate(audio, output_dir):
        with wave.open(audio) as mixture:
            frames = mixture.getnframes() if samples is None else samples
            seen.append((mixture.getframerate(), mixture.getnchannels(), mixture.getnframes()))
        written = []
        for index, amplitude in enumerate(amplitudes):
            path = os.path.join(output_dir, f"source-{index}.wav")
            with wave.open(path, "wb") as out:
                out.setnchannels(1)
                out.setsampwidth(2)
                out.setframerate(rate)
                out.writeframes(b"".join(struct.pack("<h", int(amplitude * math.sin(i / 7))) for i in range(frames)))
            written.append({"index": index, "path": path, "samples": frames})
        return {"sample_rate": rate, "sources": written if sources is None else sources}

    separate.seen = seen
    return separate


class SeparationProofTests(unittest.TestCase):
    def refused(self, separate, sources=2, rate=16000):
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_selftest.separate_synthetic(separate, sources, rate)
        self.assertEqual(caught.exception.reason, "inference_failed")

    def test_the_synthetic_mixture_is_not_a_single_tone_and_follows_the_rate_asked(self):
        with tempfile.TemporaryDirectory() as directory:
            path, tone = os.path.join(directory, "mix.wav"), os.path.join(directory, "tone.wav")
            voice_selftest.write_synthetic_mixture(path, seconds=1.0, rate=8000)
            voice_selftest.write_synthetic_audio(tone, seconds=1.0, rate=8000)
            with wave.open(path) as mixture, wave.open(tone) as single:
                self.assertEqual((mixture.getframerate(), mixture.getnchannels(), mixture.getnframes()), (8000, 1, 8000))
                self.assertNotEqual(mixture.readframes(8000), single.readframes(8000))

    def test_a_separation_that_returns_every_voice_of_the_model_passes(self):
        separate = stored_sources([9000, 4000])
        voice_selftest.separate_synthetic(separate, 2, 16000)
        self.assertEqual(separate.seen, [(16000, 1, 32000)])
        # A voice the model found nothing for may be silent, as long as one source carries signal.
        voice_selftest.separate_synthetic(stored_sources([9000, 4000, 0], rate=8000), 3, 8000)

    def test_a_wrong_number_of_sources_fails(self):
        self.refused(stored_sources([9000]))
        self.refused(stored_sources([9000, 4000, 100]))

    def test_sources_at_another_rate_or_of_another_length_fail(self):
        self.refused(stored_sources([9000, 4000], rate=8000))
        self.refused(stored_sources([9000, 4000], samples=100))

    def test_silent_sources_fail(self):
        self.refused(stored_sources([0, 0]))

    def test_a_malformed_answer_or_a_missing_file_fails(self):
        self.refused(stored_sources([9000, 4000], sources=[{"index": 0, "path": "/nonexistent/source-0.wav"}, {"index": 1, "path": "/nonexistent/source-1.wav"}]))
        self.refused(lambda audio, output_dir: {"sources": "none"})
        self.refused(lambda audio, output_dir: None)

    def test_a_library_failure_keeps_a_stable_reason_and_a_refusal_keeps_its_own(self):
        def crash(audio, output_dir):
            raise RuntimeError("CUDA error: /secret/path")
        self.refused(crash)

        def refuse(audio, output_dir):
            raise VoiceProviderUnavailable("gpu_out_of_memory")
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_selftest.separate_synthetic(refuse, 2, 16000)
        self.assertEqual(caught.exception.reason, "gpu_out_of_memory")
