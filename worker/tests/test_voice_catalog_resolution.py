"""Production preflight with catalog-only models; no ML runtime or downloads."""

import os
import tempfile
import unittest

from snitch_worker import voice, voice_models, voice_nemo


class CatalogResolutionTests(unittest.TestCase):
    def test_diarization_does_not_require_a_legacy_embedding(self):
        with tempfile.TemporaryDirectory() as root:
            env = {"SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(root, "absent")}
            repo, revision = "nvidia/sortformer", "a" * 40
            path = voice_models.model_path(env, repo, revision)
            os.makedirs(path)
            for name in ("model.nemo", voice_models.READY_MARKER):
                with open(os.path.join(path, name), "w") as handle:
                    handle.write(revision)
            provider = voice_nemo.NemoProvider(environ=env)
            voice.ensure_provider_ready(
                {"config": {"provider": "nemo", "model": repo, "model_version": revision}},
                {"nemo": provider},
                step="voice.diarization.0",
            )



class CatalogRegressionTests(unittest.TestCase):
    def install(self, env, repo, revision, name="weights.bin"):
        path = voice_models.model_path(env, repo, revision)
        os.makedirs(path)
        for filename in (name, voice_models.READY_MARKER):
            with open(os.path.join(path, filename), "w") as handle:
                handle.write(revision)
        return path

    def test_explicit_volume_does_not_depend_on_embedding_environment(self):
        with tempfile.TemporaryDirectory() as root:
            env = {"SNITCH_WORKER_MODELS_DIR": root}
            self.assertEqual(voice_models.models_root(env), os.path.join(root, "catalog"))

    def test_fingerprint_uses_its_pinned_catalog_model_without_diarization(self):
        from snitch_worker.voice_pyannote import PyannoteProvider
        with tempfile.TemporaryDirectory() as root:
            env = {"SNITCH_WORKER_MODELS_DIR": root}
            repo, revision = "synthetic/fingerprint", "b" * 40
            path = self.install(env, repo, revision)
            loaded = []
            def load(location, device):
                loaded.append((location, device))
                return lambda _audio: None
            provider = PyannoteProvider(environ=env, load_embedder=load, device_probe=lambda: "cpu")
            config = {"provider": "pyannote", "fingerprint_model": {"repo": repo, "revision": revision, "engine": "pyannote"}}
            voice.ensure_provider_ready({"config": config}, {"pyannote": provider}, step="voice.fingerprint.0")
            provider._audio_opener(config)
            other = "c" * 40
            next_path = self.install(env, repo, other)
            config["fingerprint_model"]["revision"] = other
            provider._audio_opener(config)
            self.assertEqual(loaded, [(path, "cpu"), (next_path, "cpu")])

    def test_pinned_missing_model_never_uses_an_unrelated_legacy_directory(self):
        from snitch_worker.voice_pyannote import PyannoteProvider
        from snitch_worker.voice import VoiceProviderUnavailable
        with tempfile.TemporaryDirectory() as root:
            env = {"SNITCH_WORKER_MODELS_DIR": root, "SNITCH_VOICE_DIARIZATION_MODEL_PATH": root}
            with self.assertRaises(VoiceProviderUnavailable):
                PyannoteProvider(environ=env).ensure_diarization_ready({"model": "synthetic/missing", "model_version": "d" * 40})

    def test_references_reject_trailing_newlines(self):
        self.assertFalse(voice_models.valid_model("synthetic/model\n", "a" * 40))
        self.assertFalse(voice_models.valid_model("synthetic/model", "a" * 40 + "\n"))

    def test_declarations_use_the_selected_revision_not_the_first_installed(self):
        from snitch_worker.declared_models import declared_models
        with tempfile.TemporaryDirectory() as root:
            env = {"SNITCH_WORKER_MODELS_DIR": root}
            repo = "synthetic/fingerprint"
            self.install(env, repo, "a" * 40)
            self.install(env, repo, "b" * 40)
            references = {"fingerprint": {"repo": repo, "revision": "b" * 40}}
            declared = declared_models(env, references=references).to_dict()
            self.assertEqual(declared["fingerprint"]["version"], "b" * 12)
            references["fingerprint"]["revision"] = "c" * 40
            self.assertNotIn("fingerprint", declared_models(env, references=references).to_dict())


if __name__ == "__main__":
    unittest.main()
