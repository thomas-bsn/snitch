"""Sortformer (NeMo) adapter: same result shape as pyannote, with a fake backend injected. No real model."""

import os
import sys
import tempfile
import unittest
from unittest.mock import patch

from snitch_worker import declared_models, voice, voice_models, voice_nemo, voice_pyannote

REPO = "nvidia/diar_streaming_sortformer_4spk-v2.1"
REVISION = "c" * 40
CONFIG = {
    "provider": "nemo",
    "model": REPO,
    "model_version": REVISION,
    "min_fingerprint_seconds": 1.0,
    "min_quality": 0.4,
}


def nemo_env(root):
    volume = os.path.join(root, "models")
    env = {
        "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(root, "embedding"),
        "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "wespeaker-test-1",
        "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(volume, "diarization"),
    }
    os.makedirs(env["SNITCH_VOICE_EMBEDDING_MODEL_PATH"])
    return env


def active_model(env):
    """A verified copy of the Sortformer model on the models volume (archive plus ready marker)."""
    path = voice_models.model_path(env, REPO, REVISION)
    os.makedirs(path)
    open(os.path.join(path, "model.nemo"), "w").close()
    open(os.path.join(path, voice_models.READY_MARKER), "w").close()
    return path


class NemoAdapterTests(unittest.TestCase):
    def provider(self, env, segments):
        loaded = []

        def loader(path, device):
            loaded.append((path, device))
            return lambda audio: segments

        return voice_nemo.NemoProvider(environ=env, load_pipeline=loader, device_probe=lambda: "cpu"), loaded

    def test_same_speakers_and_intervals_as_the_pyannote_adapter(self):
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            path = active_model(env)
            nemo, loaded = self.provider(env, ["2.0 3.0 speaker_1", "0.0 1.5 speaker_0", "4.0 4.5 speaker_1", "5.0 5.0 speaker_0"])
            result = nemo.diarize(CONFIG, "/a.wav", 2)
        self.assertEqual(
            result,
            {
                "provider": "nemo",
                "model": REPO,
                "model_version": REVISION,
                "audio_stream_index": 2,
                "speakers": [
                    {"label": "SPEAKER_00", "turns": [{"start": 0.0, "end": 1.5}]},
                    {"label": "SPEAKER_01", "turns": [{"start": 2.0, "end": 3.0}, {"start": 4.0, "end": 4.5}]},
                ],
            },
        )
        self.assertEqual(loaded[0][0], path, "loaded from the active model's own directory")
        self.assertEqual(env["HF_HUB_OFFLINE"], "1", "offline before any library loads")

    def test_tuples_are_accepted_and_an_empty_result_is_a_success(self):
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            active_model(env)
            nemo, _ = self.provider(env, [(0.0, 1.0, "x")])
            self.assertEqual(len(nemo.diarize(CONFIG, "/a.wav", 0)["speakers"]), 1)
            empty, _ = self.provider(env, [])
            self.assertEqual(empty.diarize(CONFIG, "/a.wav", 0)["speakers"], [])

    def test_inference_failure_is_redacted(self):
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            active_model(env)

            def loader(path, device):
                def run(audio):
                    raise RuntimeError("boom /secret/path hf_token")

                return run

            nemo = voice_nemo.NemoProvider(environ=env, load_pipeline=loader, device_probe=lambda: "cpu")
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                nemo.diarize(CONFIG, "/a.wav", 0)
        self.assertEqual(caught.exception.reason, "inference_failed")
        self.assertNotIn("secret", str(caught.exception))

    def test_fingerprints_reuse_the_pyannote_embedder(self):
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            active_model(env)
            nemo, _ = self.provider(env, ["0.0 3.0 a"])
            nemo._load_embedder = lambda path, device: lambda audio_path: voice_pyannote.DecodedAudio(lambda start, end: [1.0, end - start, 2.0], 16000, 1, 16000 * 60)
            diarization = {"job_id": 7, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 3.0}]}]}
            result = nemo.fingerprint(CONFIG, diarization, "/a.wav", 0)
        self.assertEqual(result["embedding_model_version"], "wespeaker-test-1")
        self.assertGreaterEqual(len(result["fingerprints"]), 1)

    def test_the_provider_is_registered_for_the_pipeline(self):
        self.assertIn("nemo", voice.REAL_PROVIDERS)
        self.assertIn("nemo", voice.default_providers())


class NemoWithoutRuntimeTests(unittest.TestCase):
    def test_without_nemo_the_model_is_runtime_incompatible(self):
        """No NeMo importable: the real loader raises ImportError, which preparation reports as runtime_incompatible."""
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            path = active_model(env)
            nemo = voice_nemo.NemoProvider(environ=env, device_probe=lambda: "cpu")
            with patch.dict(sys.modules, {"nemo": None, "torch": None}):
                with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                    nemo.diarize(CONFIG, "/a.wav", 0)
            self.assertEqual(caught.exception.reason, "ml_libraries_missing")
            self.assertEqual(voice_models._cause_of_verification(caught.exception), "runtime_incompatible")
            self.assertTrue(voice_nemo.nemo_archive(path).endswith("model.nemo"))


class EngineDetectionTests(unittest.TestCase):
    def test_the_engine_follows_the_model_directory_and_is_declared(self):
        with tempfile.TemporaryDirectory() as root:
            env = nemo_env(root)
            path = active_model(env)
            self.assertEqual(voice_models.engine_of(path), "nemo")
            self.assertEqual(voice_models.engine_of(root), "pyannote")
            self.assertEqual(declared_models.declared_models(env, REPO).diarization.provider, "nemo")


if __name__ == "__main__":
    unittest.main()
