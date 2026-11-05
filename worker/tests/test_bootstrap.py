import io
import os
import stat
import tempfile
import unittest

from snitch_worker import bootstrap, revocation, voice_download, voice_manifest, voice_pyannote
from snitch_worker.voice import VoiceProviderUnavailable

SECRET = "hf_SECRET_TOKEN_VALUE"


class StartTests(unittest.TestCase):
    """The start of the Worker never prepares a voice model: Snitch orders them (ADR 0038)."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = self.tmp.name

    def test_start_launches_the_worker_without_token_and_offline(self):
        launched = []
        env = {"HF_TOKEN": SECRET, "HF_HUB_OFFLINE": "0", "SNITCH_WORKER_TOKEN": "w", revocation.MODELS_DIR_ENV: self.root}
        bootstrap.main(env, launched.append)
        (worker_env,) = launched
        self.assertNotIn("HF_TOKEN", worker_env)
        self.assertEqual(worker_env["HF_HUB_OFFLINE"], "1")
        self.assertEqual(worker_env["TRANSFORMERS_OFFLINE"], "1")
        self.assertEqual(worker_env["SNITCH_WORKER_TOKEN"], "w")

    def test_the_former_download_variables_start_nothing(self):
        launched = []
        env = {
            revocation.MODELS_DIR_ENV: self.root,
            "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
            "SNITCH_VOICE_CONFIRM_TERMS": "1",
            "HF_TOKEN": SECRET,
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(self.root, "diarization"),
        }
        bootstrap.main(env, launched.append)
        self.assertEqual(len(launched), 1)
        self.assertEqual(sorted(os.listdir(self.root)), [])

    def test_there_is_no_download_entry_point_left(self):
        for name in ("run_bootstrap", "models_present", "provision", "CAUSE_CODES", "DIARIZATION_REPO", "EMBEDDING_REPO"):
            self.assertFalse(hasattr(bootstrap, name), name)

    def test_former_launcher_commands_only_start_the_worker(self):
        for command in ("check", "provision"):
            launched = []
            code = bootstrap.cli([command], {revocation.MODELS_DIR_ENV: self.root, "HF_TOKEN": SECRET}, exec_worker=launched.append)
            self.assertEqual(code, 0)
            self.assertEqual(len(launched), 1)
            self.assertEqual(os.listdir(self.root), [])


class VersionFromManifestTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.embedding = os.path.join(self.tmp.name, "embedding")
        os.makedirs(self.embedding)
        self.env = {"SNITCH_VOICE_EMBEDDING_MODEL_PATH": self.embedding}

    def provider(self, env):
        return voice_pyannote.PyannoteProvider(environ=env)

    def test_version_comes_from_the_file_when_variable_absent(self):
        voice_manifest.write_manifest(self.env, {"embedding": {"repo": "r", "revision": "rev42"}})
        self.assertEqual(self.provider(self.env)._embedding_identity()[1], "rev42")

    def test_variable_wins_over_the_file(self):
        voice_manifest.write_manifest(self.env, {"embedding": {"repo": "r", "revision": "rev42"}})
        env = {**self.env, "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "manual"}
        self.assertEqual(self.provider(env)._embedding_identity()[1], "manual")

    def test_version_still_required_without_variable_or_file(self):
        with self.assertRaises(VoiceProviderUnavailable):
            self.provider(self.env)._embedding_identity()


class TokenFileTests(unittest.TestCase):
    """The installation leaves the token in a private file of the models volume (#158)."""

    TOKEN = "hf_SyntheticFixtureToken0123456789"

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = self.tmp.name
        self.env = {
            revocation.MODELS_DIR_ENV: self.root,
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(self.root, "diarization"),
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.root, "embedding"),
        }
        self.logs = []
        self.path = os.path.join(self.root, ".secrets", "hf_token")

    def read(self):
        with open(self.path, encoding="utf-8") as handle:
            return handle.read()

    def test_store_token_reads_the_token_on_standard_input(self):
        out = io.StringIO()
        code = bootstrap.cli(["store-token"], self.env, stdin=io.StringIO(self.TOKEN + "\n"), stdout=out)
        self.assertEqual(code, 0)
        self.assertEqual(self.read(), self.TOKEN)
        self.assertEqual(stat.S_IMODE(os.stat(self.path).st_mode), 0o600)
        self.assertNotIn(self.TOKEN, out.getvalue())

    def test_store_token_refuses_an_unexpected_value_and_keeps_the_file(self):
        bootstrap.cli(["store-token"], self.env, stdin=io.StringIO(self.TOKEN), stdout=io.StringIO())
        out = io.StringIO()
        code = bootstrap.cli(["store-token"], self.env, stdin=io.StringIO("oops\n"), stdout=out)
        self.assertEqual(code, 1)
        self.assertEqual(self.read(), self.TOKEN)
        self.assertIn("cause=invalid_token", out.getvalue())
        self.assertNotIn("oops", out.getvalue())

    def test_a_deleted_worker_keeps_no_token(self):
        with open(os.path.join(self.root, revocation.MARKER_NAME), "w", encoding="utf-8") as handle:
            handle.write("revoked")
        out = io.StringIO()
        code = bootstrap.cli(["store-token"], self.env, stdin=io.StringIO(self.TOKEN), stdout=out)
        self.assertEqual(code, 1)
        self.assertIn("cause=worker_revoked", out.getvalue())
        self.assertFalse(os.path.exists(self.path))

    def test_downloading_does_not_load_the_api_client(self):
        import subprocess
        import sys

        code = "import sys, snitch_worker.voice_download; sys.exit(int('snitch_worker.revocation' in sys.modules))"
        self.assertEqual(subprocess.run([sys.executable, "-c", code]).returncode, 0)

    def test_the_default_path_lives_in_the_models_volume(self):
        self.assertEqual(voice_download.default_token_file(self.env), self.path)
        self.assertEqual(voice_download.default_token_file({}), "/models/.secrets/hf_token")
        explicit = {**self.env, "SNITCH_HF_TOKEN_FILE": "/run/secrets/hf"}
        self.assertEqual(voice_download.default_token_file(explicit), "/run/secrets/hf")


class GpuDetectionTests(unittest.TestCase):
    def test_wsl2_dxg_without_nvidiactl_counts_when_torch_is_absent(self):
        self.assertTrue(bootstrap._gpu_available(exists=lambda p: p == "/dev/dxg", torch_cuda=lambda: None))

    def test_no_device_node_and_no_torch_is_unavailable(self):
        self.assertFalse(bootstrap._gpu_available(exists=lambda _p: False, torch_cuda=lambda: None))

    def test_torch_saying_cuda_is_usable_wins_without_any_device_node(self):
        self.assertTrue(bootstrap._gpu_available(exists=lambda _p: False, torch_cuda=lambda: True))

    def test_torch_saying_cuda_is_unusable_rejects_despite_device_nodes(self):
        self.assertFalse(bootstrap._gpu_available(exists=lambda _p: True, torch_cuda=lambda: False))


if __name__ == "__main__":
    unittest.main()
