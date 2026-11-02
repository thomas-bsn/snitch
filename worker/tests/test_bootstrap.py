import io
import os
import stat
import tempfile
import unittest

from snitch_worker import bootstrap, revocation, voice_download, voice_manifest, voice_pyannote
from snitch_worker.voice import VoiceProviderUnavailable

SECRET = "hf_SECRET_TOKEN_VALUE"


class Fakes:
    def __init__(self, fail=None):
        self.downloads = []
        self.fail = fail

    def downloader(self, repo_id, local_dir, token, revision):
        self.downloads.append((repo_id, local_dir, token, revision))
        if self.fail:
            raise self.fail
        os.makedirs(local_dir, exist_ok=True)
        with open(os.path.join(local_dir, "weights.bin"), "w") as handle:
            handle.write("x")

    def resolve(self, repo_id, revision, token):
        return "sha-" + repo_id.split("/")[-1][:4]


class BootstrapTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = self.tmp.name
        self.env = {
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(self.root, "diarization"),
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.root, "embedding"),
        }
        self.logs = []

    def run_bootstrap(self, fakes, **extra):
        env = {**self.env, **extra}
        return bootstrap.run_bootstrap(env, fakes.downloader, fakes.resolve, self.logs.append)

    def optin(self, **extra):
        return {
            "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
            "SNITCH_VOICE_CONFIRM_TERMS": "1",
            "HF_TOKEN": SECRET,
            **extra,
        }

    def test_without_opt_in_nothing_is_downloaded(self):
        fakes = Fakes()
        self.assertEqual(self.run_bootstrap(fakes, HF_TOKEN=SECRET), "disabled")
        self.assertEqual(fakes.downloads, [])

    def test_opt_in_without_terms_confirmation_downloads_nothing(self):
        fakes = Fakes()
        status = self.run_bootstrap(fakes, SNITCH_VOICE_DOWNLOAD_ON_START="1", HF_TOKEN=SECRET)
        self.assertEqual(status, "terms_not_confirmed")
        self.assertEqual(fakes.downloads, [])
        self.assertIn("SNITCH_VOICE_CONFIRM_TERMS", "\n".join(self.logs))

    def test_opt_in_without_token_downloads_nothing_and_says_why(self):
        fakes = Fakes()
        status = self.run_bootstrap(fakes, SNITCH_VOICE_DOWNLOAD_ON_START="1", SNITCH_VOICE_CONFIRM_TERMS="1")
        self.assertEqual(status, "token_missing")
        self.assertEqual(fakes.downloads, [])
        self.assertIn("HF_TOKEN", "\n".join(self.logs))

    def test_with_token_downloads_both_models_and_writes_marker(self):
        fakes = Fakes()
        self.assertEqual(self.run_bootstrap(fakes, **self.optin()), "downloaded")
        repos = [d[0] for d in fakes.downloads]
        self.assertEqual(repos, [bootstrap.DIARIZATION_REPO, bootstrap.EMBEDDING_REPO])
        self.assertTrue(all(d[2] == SECRET for d in fakes.downloads))
        self.assertEqual(voice_manifest.read_embedding_version(self.env), "sha-wesp")
        self.assertNotIn(SECRET, "\n".join(self.logs))
        with open(voice_manifest.manifest_path(self.env)) as handle:
            self.assertNotIn(SECRET, handle.read())

    def test_models_already_present_trigger_no_call_even_without_token(self):
        self.run_bootstrap(Fakes(), **self.optin())
        again = Fakes()
        status = self.run_bootstrap(again, SNITCH_VOICE_DOWNLOAD_ON_START="1", SNITCH_VOICE_CONFIRM_TERMS="1")
        self.assertEqual(status, "present")
        self.assertEqual(again.downloads, [])

    def test_missing_model_files_invalidate_the_marker(self):
        self.run_bootstrap(Fakes(), **self.optin())
        os.remove(os.path.join(self.env["SNITCH_VOICE_EMBEDDING_MODEL_PATH"], "weights.bin"))
        again = Fakes()
        self.assertEqual(self.run_bootstrap(again, **self.optin()), "downloaded")
        self.assertEqual(len(again.downloads), 2)

    def test_pinned_revision_different_from_marker_downloads_again(self):
        self.run_bootstrap(Fakes(), **self.optin())
        again = Fakes()
        status = self.run_bootstrap(again, **self.optin(SNITCH_VOICE_MODEL_REVISION="abc123"))
        self.assertEqual(status, "downloaded")
        self.assertTrue(all(d[3] == "abc123" for d in again.downloads))
        self.assertEqual(voice_manifest.read_embedding_version(self.env), "abc123")

    def test_failure_is_redacted_and_leaves_no_marker(self):
        fakes = Fakes(fail=OSError("401 Bearer " + SECRET + " https://hf.co/x?token=" + SECRET))
        status = self.run_bootstrap(fakes, **self.optin())
        self.assertEqual(status, "failed")
        self.assertNotIn(SECRET, "\n".join(self.logs))
        self.assertIsNone(voice_manifest.read_manifest(self.env))

    def test_failure_causes_are_readable(self):
        class GatedRepoError(Exception):
            pass

        self.run_bootstrap(Fakes(fail=GatedRepoError(SECRET)), **self.optin())
        self.assertIn("terms", "\n".join(self.logs))
        self.logs.clear()
        self.run_bootstrap(Fakes(fail=ConnectionError(SECRET)), **self.optin())
        self.assertIn("network", "\n".join(self.logs))
        self.assertNotIn(SECRET, "\n".join(self.logs))

    def test_main_launches_worker_without_token_and_offline(self):
        launched = []
        env = {**self.env, "HF_TOKEN": SECRET, "HF_HUB_OFFLINE": "0", "SNITCH_WORKER_TOKEN": "w"}
        fakes = Fakes()
        bootstrap.main(env, fakes.downloader, fakes.resolve, launched.append, self.logs.append)
        (worker_env,) = launched
        self.assertNotIn("HF_TOKEN", worker_env)
        self.assertEqual(worker_env["HF_HUB_OFFLINE"], "1")
        self.assertEqual(worker_env["TRANSFORMERS_OFFLINE"], "1")
        self.assertEqual(worker_env["SNITCH_WORKER_TOKEN"], "w")
        self.assertEqual(fakes.downloads, [])

    def test_main_still_launches_worker_when_download_fails(self):
        launched = []
        env = {**self.env, **self.optin()}
        fakes = Fakes(fail=OSError(SECRET))
        bootstrap.main(env, fakes.downloader, fakes.resolve, launched.append, self.logs.append)
        self.assertEqual(len(launched), 1)
        self.assertNotIn("HF_TOKEN", launched[0])

    def test_unexpected_bootstrap_error_does_not_stop_worker(self):
        launched = []

        def boom(*args):
            raise RuntimeError(SECRET)

        bootstrap.main({**self.env, **self.optin()}, Fakes().downloader, boom, launched.append, self.logs.append)
        self.assertEqual(len(launched), 1)
        self.assertNotIn(SECRET, "\n".join(self.logs))

    def test_permission_error_on_models_directory_is_reported_cleanly(self):
        # Simulate a directory the container user cannot write to
        import stat
        readonly_root = tempfile.TemporaryDirectory()
        self.addCleanup(readonly_root.cleanup)
        diarization_path = os.path.join(readonly_root.name, "diarization")
        embedding_path = os.path.join(readonly_root.name, "embedding")
        os.makedirs(diarization_path, exist_ok=True)
        os.makedirs(embedding_path, exist_ok=True)
        # Make the parent directory read-only (simulating a host mount with wrong permissions)
        os.chmod(readonly_root.name, stat.S_IRUSR | stat.S_IXUSR)  # 0500 - no write
        try:
            env = {
                **self.env,
                **self.optin(),
                "SNITCH_VOICE_DIARIZATION_MODEL_PATH": diarization_path,
                "SNITCH_VOICE_EMBEDDING_MODEL_PATH": embedding_path,
            }
            status = bootstrap.run_bootstrap(env, Fakes().downloader, Fakes().resolve, self.logs.append)
            self.assertEqual(status, "failed")
            self.assertIn("models directory is not writable", "\n".join(self.logs))
            self.assertNotIn(SECRET, "\n".join(self.logs))
        finally:
            # Restore permissions for cleanup
            os.chmod(readonly_root.name, stat.S_IRWXU)


class StableCauseTests(unittest.TestCase):
    """Cause codes are the closed vocabulary shared with the launcher and the events contract."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.env = {
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(self.tmp.name, "d"),
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.tmp.name, "e"),
            "HF_HOME": "/models/.cache/huggingface",
            "HF_XET_CACHE": "/models/.cache/huggingface/xet",
            "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
            "SNITCH_VOICE_CONFIRM_TERMS": "1",
            "HF_TOKEN": SECRET,
        }

    def code_for(self, error, env=None):
        logs = []
        report = {}
        status = bootstrap.run_bootstrap(env or self.env, Fakes(fail=error).downloader, Fakes().resolve, logs.append, report)
        self.assertEqual(status, "failed")
        self.assertNotIn(SECRET, "\n".join(logs))
        self.assertIn(report["cause_code"], bootstrap.CAUSE_CODES)
        return report["cause_code"]

    def test_gated_repo_and_http_denials_are_hf_access_denied(self):
        class GatedRepoError(Exception):
            pass

        class Resp:
            status_code = 401

        class HTTPError(Exception):
            response = Resp()

        self.assertEqual(self.code_for(GatedRepoError(SECRET)), "hf_access_denied")
        self.assertEqual(self.code_for(HTTPError(SECRET)), "hf_access_denied")

    def test_network_errors(self):
        self.assertEqual(self.code_for(ConnectionError(SECRET)), "network_unreachable")

    def test_models_volume_vs_hf_cache_permission(self):
        models = PermissionError(13, "denied", "/models/diarization/x")
        cache = PermissionError(13, "denied", "/models/.cache/huggingface/xet/chunk")
        self.assertEqual(self.code_for(models), "volume_not_writable")
        self.assertEqual(self.code_for(cache), "hf_cache_not_writable")

    def test_unknown_error_has_unknown_code_and_no_text(self):
        self.assertEqual(self.code_for(RuntimeError(SECRET)), "unknown")

    def test_missing_gpu_is_gpu_unavailable(self):
        self.assertEqual(self.code_for(RuntimeError("CUDA error: no CUDA-capable device " + SECRET)), "unknown")
        report = {}
        self.assertEqual(
            bootstrap.run_bootstrap({**self.env, "SNITCH_VOICE_REQUIRE_GPU": "1"}, Fakes().downloader, Fakes().resolve, lambda _l: None, report, gpu_available=lambda: False),
            "failed",
        )
        self.assertEqual(report["cause_code"], "gpu_unavailable")

    def test_cli_provision_prints_only_the_code(self):
        import io
        from contextlib import redirect_stdout

        out = io.StringIO()
        with redirect_stdout(out):
            rc = bootstrap.cli(["provision"], self.env, Fakes(fail=ConnectionError(SECRET)).downloader, Fakes().resolve, lambda _l: None)
        self.assertEqual(rc, 1)
        self.assertEqual(out.getvalue().strip(), "cause=network_unreachable")

    def test_cli_check_reflects_manifest(self):
        self.assertEqual(bootstrap.cli(["check"], self.env), 1)
        bootstrap.run_bootstrap(self.env, Fakes().downloader, Fakes().resolve, lambda _l: None)
        self.assertEqual(bootstrap.cli(["check"], self.env), 0)


class ResumeTests(unittest.TestCase):
    def test_partial_download_resumes_in_place_and_never_wipes_files(self):
        with tempfile.TemporaryDirectory() as root:
            env = {
                "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(root, "d"),
                "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(root, "e"),
                "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
                "SNITCH_VOICE_CONFIRM_TERMS": "1",
                "HF_TOKEN": SECRET,
            }
            seen = []

            def flaky(repo_id, local_dir, token, revision):
                os.makedirs(local_dir, exist_ok=True)
                part = os.path.join(local_dir, "weights.bin.incomplete")
                seen.append(os.path.exists(part))
                if len(seen) == 1:
                    with open(part, "w") as handle:
                        handle.write("half")
                    raise ConnectionError(SECRET)
                with open(os.path.join(local_dir, "weights.bin"), "w") as handle:
                    handle.write("x")

            resolve = Fakes().resolve
            logs = []
            self.assertEqual(bootstrap.run_bootstrap(env, flaky, resolve, logs.append), "failed")
            self.assertIsNone(voice_manifest.read_manifest(env))  # partial is never "ready"
            self.assertFalse(bootstrap.models_present(env))
            self.assertEqual(bootstrap.run_bootstrap(env, flaky, resolve, logs.append), "downloaded")
            self.assertTrue(seen[1], "the second run must start from the partial file, not wipe it")
            self.assertNotIn(SECRET, "\n".join(logs))


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
            "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
            "SNITCH_VOICE_CONFIRM_TERMS": "1",
        }
        self.logs = []
        self.path = os.path.join(self.root, ".secrets", "hf_token")

    def start(self, **extra):
        fakes = Fakes()
        status = bootstrap.run_bootstrap({**self.env, **extra}, fakes.downloader, fakes.resolve, self.logs.append)
        return status

    def read(self):
        with open(self.path, encoding="utf-8") as handle:
            return handle.read()

    def test_the_token_given_at_installation_is_kept_in_a_private_file(self):
        self.assertEqual(self.start(HF_TOKEN=self.TOKEN), "downloaded")
        self.assertEqual(self.read(), self.TOKEN)
        self.assertEqual(stat.S_IMODE(os.stat(self.path).st_mode), 0o600)
        self.assertEqual(stat.S_IMODE(os.stat(os.path.dirname(self.path)).st_mode), 0o700)
        self.assertNotIn(self.TOKEN, "\n".join(self.logs))

    def test_models_already_present_still_keep_the_token_for_later_installs(self):
        self.assertEqual(self.start(HF_TOKEN=self.TOKEN), "downloaded")
        os.remove(self.path)
        self.assertEqual(self.start(HF_TOKEN=self.TOKEN), "present")
        self.assertEqual(self.read(), self.TOKEN)

    def test_without_the_opt_in_nothing_is_stored(self):
        self.assertEqual(self.start(HF_TOKEN=self.TOKEN, SNITCH_VOICE_DOWNLOAD_ON_START="0"), "disabled")
        self.assertFalse(os.path.exists(self.path))

    def test_an_unexpected_token_never_replaces_the_kept_one(self):
        self.start(HF_TOKEN=self.TOKEN)
        self.start(HF_TOKEN="not a token")
        self.assertEqual(self.read(), self.TOKEN)
        self.assertNotIn("not a token", "\n".join(self.logs))

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

    def test_the_default_path_lives_in_the_models_volume(self):
        self.assertEqual(voice_download.default_token_file(self.env), self.path)
        self.assertEqual(voice_download.default_token_file({}), "/models/.secrets/hf_token")
        explicit = {**self.env, "SNITCH_HF_TOKEN_FILE": "/run/secrets/hf"}
        self.assertEqual(voice_download.default_token_file(explicit), "/run/secrets/hf")


if __name__ == "__main__":
    unittest.main()
