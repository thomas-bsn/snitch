"""The Worker's voice model manager: prepares the desired models, reports honestly, never leaks the token."""

import os
import tempfile
import unittest
from unittest.mock import Mock

from snitch_worker import declared_models, voice_download, voice_models
from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_pyannote import PyannoteProvider

REPO = "synthetic/diarization-b"
REVISION = "b" * 40
MODEL = {"model_id": "test-b", "engine": "pyannote", "repo": REPO, "revision": REVISION, "role": "candidate", "retry": False}
SECRET = "hf_synthetic_secret_value_0123456789"


class _NoTurns:
    def itertracks(self, yield_label=False):
        return []


class Harness:
    """A manager with a fake download, a fake inference proof and a recorded report."""

    def __init__(self, test: unittest.TestCase, download=None, verify=None):
        self.directory = tempfile.TemporaryDirectory()
        test.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.directory.name, "embedding")}
        self.reports: list[dict] = []
        self.downloads: list[tuple[str, str, str]] = []
        self.verified: list[tuple[str, dict]] = []

        def default_download(repo, revision, target, progress):
            self.downloads.append((repo, revision, target))
            progress(10_000_000, 30_000_000)
            os.makedirs(target, exist_ok=True)
            with open(os.path.join(target, "weights.bin"), "w") as handle:
                handle.write("synthetic")
            progress(30_000_000, 30_000_000)

        def default_verify(path, environ):
            self.verified.append((path, dict(environ)))

        self.manager = voice_models.ModelManager(
            self.env, download or default_download, verify or default_verify, self.reports.append, min_progress_interval=0
        )

    def sync(self, models=(), remove=()):
        self.manager.apply({"models": list(models), "remove": list(remove)})
        self.manager.wait()

    def states(self, model_id="test-b"):
        return [r["state"] for r in self.reports if r["model_id"] == model_id]


class PreparationTests(unittest.TestCase):
    def test_candidate_goes_through_download_verification_and_ready(self):
        h = Harness(self)
        h.sync([MODEL])
        self.assertEqual(h.states(), ["downloading", "downloading", "downloading", "verifying", "ready"])
        progress = [(r["bytes_received"], r["bytes_total"]) for r in h.reports if r["state"] == "downloading"]
        self.assertIn((10_000_000, 30_000_000), progress)
        self.assertEqual(h.reports[-1]["cause_code"], "")
        self.assertEqual(len(h.downloads), 1)
        self.assertEqual(h.verified[0][0], h.downloads[0][2])

    def test_a_ready_model_is_not_prepared_again(self):
        h = Harness(self)
        h.sync([MODEL])
        reports = len(h.reports)
        h.sync([MODEL])
        self.assertEqual(len(h.downloads), 1)
        self.assertEqual(len(h.reports), reports)

    def test_a_restart_keeps_the_proof_without_downloading_again(self):
        h = Harness(self)
        h.sync([MODEL])
        restarted: list[dict] = []
        second = voice_models.ModelManager(h.env, h.manager._download, h.manager._verify, restarted.append)
        second.apply({"models": [MODEL], "remove": []})
        second.wait()
        self.assertEqual(len(h.downloads), 1)
        self.assertEqual([r["state"] for r in restarted], ["ready"])

    def test_an_interrupted_download_resumes_in_the_same_directory(self):
        h = Harness(self)
        target = voice_models.model_path(h.env, REPO, REVISION)
        os.makedirs(target)
        open(os.path.join(target, "partial.bin"), "w").close()
        h.sync([MODEL])
        self.assertEqual(h.downloads[0][2], target)
        self.assertEqual(h.states()[-1], "ready")

    def test_failures_carry_a_stable_cause_and_are_not_retried_alone(self):
        for error, cause in (
            (voice_models.DownloadFailed("hf_token_missing"), "hf_token_missing"),
            (voice_models.DownloadFailed("hf_access_denied"), "hf_access_denied"),
            (OSError(28, "No space left on device"), "disk_full"),
            (RuntimeError(f"boom {SECRET}"), "runtime_incompatible"),
        ):
            def failing(repo, revision, target, progress, error=error):
                raise error

            h = Harness(self, download=failing)
            h.sync([MODEL])
            self.assertEqual(h.reports[-1], {"model_id": "test-b", "state": "failed", "cause_code": cause, "bytes_received": 0, "bytes_total": 0})
            self.assertNotIn(SECRET, repr(h.reports))
            h.sync([MODEL])
            self.assertEqual(sum(1 for r in h.reports if r["state"] == "failed"), 1)

    def test_retry_prepares_again_after_a_failure(self):
        calls = {"n": 0}

        def flaky(repo, revision, target, progress):
            calls["n"] += 1
            if calls["n"] == 1:
                raise voice_models.DownloadFailed("hf_access_denied")
            os.makedirs(target, exist_ok=True)
            open(os.path.join(target, "weights.bin"), "w").close()

        h = Harness(self, download=flaky)
        h.sync([MODEL])
        self.assertEqual(h.states()[-1], "failed")
        h.sync([{**MODEL, "retry": True}])
        self.assertEqual(h.states()[-1], "ready")

    def test_verification_failures_map_to_runtime_or_inference_causes(self):
        for reason, cause in (("model_load_failed", "runtime_incompatible"), ("ml_libraries_missing", "runtime_incompatible"), ("inference_failed", "inference_failed")):
            def verify(path, environ, reason=reason):
                raise VoiceProviderUnavailable(reason)

            h = Harness(self, verify=verify)
            h.sync([MODEL])
            self.assertEqual(h.states()[-3:], ["downloading", "verifying", "failed"])
            self.assertEqual(h.reports[-1]["cause_code"], cause)
            # A downloaded but unverified model is never ready.
            self.assertFalse(voice_models.is_ready(h.env, REPO, REVISION))

    def test_the_inference_process_never_sees_the_token(self):
        h = Harness(self)
        h.env.update({"HF_TOKEN": SECRET, "SNITCH_HF_TOKEN_FILE": "/run/secrets/hf"})
        h.sync([MODEL])
        _, environ = h.verified[0]
        self.assertNotIn("HF_TOKEN", environ)
        self.assertNotIn("SNITCH_HF_TOKEN_FILE", environ)
        self.assertNotIn(SECRET, repr(environ) + repr(h.reports))
        self.assertEqual(environ["HF_HUB_OFFLINE"], "1")

    def test_only_catalogue_shaped_models_are_prepared(self):
        h = Harness(self)
        h.sync([{**MODEL, "repo": "../../etc", "model_id": "evil"}, {**MODEL, "revision": "../x", "model_id": "evil2"}])
        self.assertEqual(h.downloads, [])
        self.assertEqual(h.reports, [])


class RemovalTests(unittest.TestCase):
    def test_removal_deletes_the_model_and_reports_it(self):
        h = Harness(self)
        h.sync([MODEL])
        path = voice_models.model_path(h.env, REPO, REVISION)
        self.assertTrue(os.path.isdir(path))
        h.sync([], [MODEL])
        self.assertFalse(os.path.exists(path))
        self.assertEqual(h.reports[-1]["state"], "removed")
        self.assertFalse(voice_models.is_ready(h.env, REPO, REVISION))


class DownloadProcessTests(unittest.TestCase):
    """The download subprocess is the only place that reads the token file."""

    def run_download(self, token_file_content, snapshot):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        env = {}
        if token_file_content is not None:
            path = os.path.join(directory.name, "hf-token")
            with open(path, "w") as handle:
                handle.write(token_file_content)
            env["SNITCH_HF_TOKEN_FILE"] = path
        lines: list[str] = []
        code = voice_download.run(REPO, REVISION, os.path.join(directory.name, "target"), env, snapshot, lambda *_: 30_000_000, lines.append)
        return code, lines

    def test_a_missing_token_file_fails_before_any_network_call(self):
        snapshot = Mock()
        for content in (None, "", "  \n"):
            code, lines = self.run_download(content, snapshot)
            self.assertEqual(code, 1)
            self.assertIn('"hf_token_missing"', lines[-1])
        snapshot.assert_not_called()

    def test_the_token_goes_to_the_download_and_nowhere_else(self):
        seen = {}

        def snapshot(repo, revision, target, token, on_progress):
            seen["token"] = token
            on_progress(5, 30_000_000)

        code, lines = self.run_download(SECRET + "\n", snapshot)
        self.assertEqual(code, 0)
        self.assertEqual(seen["token"], SECRET)
        self.assertNotIn(SECRET, "".join(lines))
        self.assertIn('"done"', lines[-1])

    def test_errors_become_stable_causes_without_library_text(self):
        class GatedRepoError(Exception):
            pass

        for error, cause in ((GatedRepoError(f"denied {SECRET}"), "hf_access_denied"), (OSError(28, "full"), "disk_full"), (ValueError(SECRET), "runtime_incompatible")):
            def snapshot(*_args, error=error):
                raise error

            code, lines = self.run_download(SECRET, snapshot)
            self.assertEqual(code, 1)
            self.assertIn(f'"{cause}"', lines[-1])
            self.assertNotIn(SECRET, "".join(lines))


class SyncTests(unittest.TestCase):
    def client(self, status, body):
        client = Mock()
        http = Mock()
        client.get_httpx_client.return_value = http
        response = Mock(status_code=status)
        response.json.return_value = body
        http.get.return_value = response
        return client, http

    def test_sync_pulls_the_desired_state_and_posts_each_report(self):
        h = Harness(self)
        client, http = self.client(200, {"models": [MODEL], "remove": []})
        http.post.return_value = Mock(status_code=204)
        voice_models.sync_voice_models(client, h.manager)
        h.manager.wait()
        http.get.assert_called_with("/api/worker/voice-models")
        posted = [call.kwargs["json"]["state"] for call in http.post.call_args_list]
        self.assertEqual(posted[-2:], ["verifying", "ready"])
        self.assertEqual(voice_models.active_repo(), "")

    def test_a_server_without_the_contract_is_tolerated(self):
        h = Harness(self)
        client, http = self.client(404, {})
        voice_models.sync_voice_models(client, h.manager)
        self.assertEqual(h.reports, [])


class UsageTests(unittest.TestCase):
    def test_the_worker_declares_the_ready_active_model(self):
        h = Harness(self)
        h.sync([MODEL])
        env = {**h.env}
        declared = declared_models.declared_models(env, active_diarization=REPO).to_dict()
        self.assertEqual(declared["diarization"]["model"], REPO)
        self.assertEqual(declared["diarization"]["version"], REVISION[:12])
        self.assertNotIn("diarization", declared_models.declared_models(env, active_diarization="other/model").to_dict())

    def test_inference_loads_the_active_models_own_directory(self):
        h = Harness(self)
        h.sync([MODEL])
        os.makedirs(h.env["SNITCH_VOICE_EMBEDDING_MODEL_PATH"], exist_ok=True)
        loaded = []
        provider = PyannoteProvider(
            environ={**h.env, "SNITCH_VOICE_DIARIZATION_MODEL_PATH": h.directory.name, "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "v"},
            load_pipeline=lambda path, device: loaded.append(path) or (lambda audio: _NoTurns()),
            device_probe=lambda: "cpu",
        )
        provider.ensure_ready({"model": REPO})
        provider.diarize({"provider": "pyannote", "model": REPO, "model_version": REVISION}, "x.wav", 0)
        self.assertEqual(loaded, [voice_models.model_path(h.env, REPO, REVISION)])


if __name__ == "__main__":
    unittest.main()
