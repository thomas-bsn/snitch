import unittest

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
