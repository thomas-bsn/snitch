"""The Worker's voice model manager: prepares the desired models, reports honestly, never leaks the token."""

import io
import json
import logging
import os
import tempfile
import unittest
from unittest.mock import Mock, patch

from snitch_worker import declared_models, log, voice_download, voice_models
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
        # What an order does: the manager only prepares a model it is explicitly handed.
        for item in models:
            self.manager.prepare(item, self.reports.append)
        self.manager.remove(list(remove), self.reports.append)

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
        # An explicit order for a model already held answers "ready" once, without a download or a verification.
        self.assertEqual([r["state"] for r in h.reports[reports:]], ["ready"])
        self.assertEqual(len(h.verified), 1)

    def test_a_restart_keeps_the_proof_without_downloading_again(self):
        h = Harness(self)
        h.sync([MODEL])
        restarted: list[dict] = []
        second = voice_models.ModelManager(h.env, h.manager._download, h.manager._verify, restarted.append)
        second.prepare(MODEL, restarted.append)
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

    def test_failures_carry_a_stable_cause(self):
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

    def test_a_new_order_prepares_again_after_a_failure(self):
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
        h.sync([MODEL])
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


class TokenHygieneTests(unittest.TestCase):
    def test_the_manager_takes_the_token_file_out_of_the_process_environment(self):
        env = {"SNITCH_HF_TOKEN_FILE": "/run/secrets/hf", "HF_TOKEN": SECRET, "PATH": "/bin"}
        with patch.dict(os.environ, env, clear=True), patch.object(voice_models, "_manager", None):
            voice_models.default_manager()
            self.assertNotIn("SNITCH_HF_TOKEN_FILE", os.environ)
            # Only the download subprocess gets the path, explicitly, and never the token variables.
            download = voice_models.download_environment("/run/secrets/hf")
            self.assertEqual(download["SNITCH_HF_TOKEN_FILE"], "/run/secrets/hf")
            self.assertNotIn("HF_TOKEN", download)
            self.assertNotIn("HUGGING_FACE_HUB_TOKEN", download)
            self.assertEqual(download["HF_HUB_OFFLINE"], "0")
            self.assertNotIn("SNITCH_HF_TOKEN_FILE", voice_models.download_environment(""))

    def test_without_a_variable_the_manager_downloads_with_the_token_kept_in_the_models_volume(self):
        seen = []
        env = {"SNITCH_WORKER_MODELS_DIR": "/data", "PATH": "/bin"}
        with (
            patch.dict(os.environ, env, clear=True),
            patch.object(voice_models, "_manager", None),
            patch.object(voice_models, "_subprocess_download", lambda *args: seen.append(args[-1])),
        ):
            voice_models.default_manager()._download("owner/model", "rev", "/data/target", lambda received, total: None)
        self.assertEqual(seen, ["/data/.secrets/hf_token"])

    def test_one_filter_serves_every_environment(self):
        env = {"HF_TOKEN": "a", "HUGGING_FACE_HUB_TOKEN": "b", "SNITCH_HF_TOKEN_FILE": "/p", "X": "1"}
        self.assertEqual(voice_download.without_token(env), {"X": "1"})
        self.assertEqual(voice_download.without_token(env, keep_file=True), {"SNITCH_HF_TOKEN_FILE": "/p", "X": "1"})
        self.assertIs(voice_models.TOKEN_FILE_ENV, voice_download.TOKEN_FILE_ENV)


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

        for error, cause in ((GatedRepoError(f"denied {SECRET}"), "hf_auth_required"), (OSError(28, "full"), "disk_full"), (ValueError(SECRET), "runtime_incompatible")):
            def snapshot(*_args, error=error):
                raise error

            code, lines = self.run_download(SECRET, snapshot)
            self.assertEqual(code, 1)
            self.assertIn(f'"{cause}"', lines[-1])
            self.assertNotIn(SECRET, "".join(lines))


class _Http(Exception):
    def __init__(self, status):
        super().__init__(f"{status} for https://cdn.example/signed?X-Amz-Signature=abc {SECRET}")
        self.response = Mock(status_code=status)


class HuggingFaceFailureFamilyTests(unittest.TestCase):
    """Each simulated failure family of the download gets its own catalogue code, never the library text."""

    def run_failing(self, error):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        path = os.path.join(directory.name, "hf-token")
        with open(path, "w") as handle:
            handle.write(SECRET)

        def snapshot(*_args):
            raise error

        lines: list[str] = []
        code = voice_download.run(REPO, REVISION, os.path.join(directory.name, "t"), {"SNITCH_HF_TOKEN_FILE": path, "HF_HOME": "/models/.cache/huggingface"}, snapshot, lambda *_: 1, lines.append)
        self.assertEqual(code, 1)
        self.assertNotIn(SECRET, "".join(lines))
        self.assertNotIn("Signature", "".join(lines))
        return json.loads(lines[-1])["error"]

    def test_every_family_has_a_distinct_code(self):
        class ConnectError(Exception):
            pass

        env_cache = "/models/.cache/huggingface"
        cases = [
            (_Http(401), "hf_auth_required"),
            (_Http(403), "hf_auth_required"),
            (_Http(404), "hf_not_found"),
            (ConnectError(SECRET), "hf_network"),
            (TimeoutError(SECRET), "hf_network"),
            (PermissionError(13, "denied", "/models/catalog/x/y/weights.bin"), "models_volume_unwritable"),
            (OSError(30, "read-only", "/models/catalog/x"), "models_volume_unwritable"),
            (PermissionError(13, "denied", env_cache + "/xet/chunk"), "hf_cache_error"),
        ]
        for error, expected in cases:
            self.assertEqual(self.run_failing(error), expected, repr(error))


def _logged(test, level="DEBUG"):
    stream = io.StringIO()
    log.configure(level, stream=stream)
    test.addCleanup(log.configure, "INFO", io.StringIO())
    return lambda: [json.loads(line) for line in stream.getvalue().splitlines() if line.strip()]


class DownloadMilestoneTests(unittest.TestCase):
    def events(self, lines, name):
        return [e for e in lines if e["event"] == name]

    def test_a_successful_activation_leaves_the_milestones_with_the_revision(self):
        lines = _logged(self)
        h = Harness(self)
        h.sync([{**MODEL, "role": "active"}])
        names = [e["event"] for e in lines() if e["event"].startswith("voice_model_")]
        self.assertEqual([n for n in names if n != "voice_model_download_progress"], ["voice_model_download_started", "voice_model_validated", "voice_model_activated"])
        for e in lines():
            if e["event"].startswith("voice_model_"):
                self.assertEqual((e["model_id"], e["revision"]), ("test-b", REVISION))
                self.assertEqual(e["level"], "DEBUG" if e["event"] == "voice_model_download_progress" else "INFO")

    def test_progress_is_debug_only_and_bounded_to_steps(self):
        def download(repo, revision, target, progress):
            os.makedirs(target, exist_ok=True)
            for received in range(0, 1001):
                progress(received, 1000)

        lines = _logged(self)
        Harness(self, download=download).sync([MODEL])
        progress = self.events(lines(), "voice_model_download_progress")
        self.assertLessEqual(len(progress), 5)
        self.assertEqual(progress[-1]["percent"], 100)
        info = _logged(self, "INFO")
        Harness(self, download=download).sync([MODEL])
        self.assertEqual(self.events(info(), "voice_model_download_progress"), [])

    def test_a_download_in_an_existing_directory_is_logged_as_resumed(self):
        lines = _logged(self)
        h = Harness(self)
        target = voice_models.model_path(h.env, REPO, REVISION)
        os.makedirs(target)
        open(os.path.join(target, "partial.bin"), "w").close()
        h.sync([MODEL])
        self.assertEqual(len(self.events(lines(), "voice_model_download_resumed")), 1)
        self.assertEqual(self.events(lines(), "voice_model_download_started"), [])

    def test_a_failure_logs_the_code_and_never_the_library_text(self):
        lines = _logged(self)

        def failing(repo, revision, target, progress):
            raise _Http(404)

        Harness(self, download=failing).sync([MODEL])
        [failed] = self.events(lines(), "voice_model_failed")
        self.assertEqual((failed["cause_code"], failed["level"], failed["revision"]), ("hf_not_found", "INFO", REVISION))
        self.assertNotIn(SECRET, json.dumps(lines()))
        self.assertNotIn("Signature", json.dumps(lines()))

    def test_the_launcher_reads_the_codes_of_the_download_process(self):
        seen = []
        for code in ("hf_auth_required", "hf_not_found", "hf_network", "models_volume_unwritable", "hf_cache_error", "disk_full"):
            stdout = [json.dumps({"received": 5, "total": 10}), "not json " + SECRET, json.dumps({"error": code})]
            self.assertEqual(voice_models.read_download_output(stdout, lambda r, t: seen.append((r, t))), code)
        self.assertEqual(voice_models.read_download_output([json.dumps({"error": "something " + SECRET})], lambda *_: None), "runtime_incompatible")
        self.assertEqual(seen[0], (5, 10))


class SyncTests(unittest.TestCase):
    def client(self, status, body):
        client = Mock()
        http = Mock()
        client.get_httpx_client.return_value = http
        response = Mock(status_code=status)
        response.json.return_value = body
        http.get.return_value = response
        return client, http

    def test_sync_never_prepares_a_model_snitch_merely_lists(self):
        h = Harness(self)
        client, http = self.client(200, {"models": [MODEL, {**MODEL, "role": "active", "model_id": "test-a"}], "remove": [], "orders": []})
        voice_models.sync_voice_models(client, h.manager)
        http.get.assert_called_with("/api/worker/voice-models")
        self.assertEqual(h.downloads, [])
        self.assertEqual(h.verified, [])
        http.post.assert_not_called()
        self.assertFalse(os.path.exists(voice_models.models_root(h.env)))

    def test_a_pull_after_a_restart_downloads_nothing_even_for_a_failed_or_missing_model(self):
        for _ in range(2):  # a fresh manager each time: nothing is remembered in the process
            h = Harness(self)
            client, http = self.client(200, {"models": [MODEL], "remove": []})
            voice_models.sync_voice_models(client, h.manager)
            self.assertEqual(h.downloads, [])

    def test_sync_still_removes_what_snitch_asks_to_remove(self):
        h = Harness(self)
        h.sync([MODEL])
        path = voice_models.model_path(h.env, REPO, REVISION)
        client, http = self.client(200, {"models": [], "remove": [MODEL]})
        http.post.return_value = Mock(status_code=204)
        voice_models.sync_voice_models(client, h.manager)
        self.assertFalse(os.path.exists(path))
        self.assertEqual(http.post.call_args.kwargs["json"]["state"], "removed")

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
