"""A separation model on the Worker (#401, ADR 0066): it is installed by an explicit order only, it is ready only
after a test inference on a synthetic mixture, a failure carries a stable cause, and separating audio with a model
that is absent or not ready is an explicit state that downloads nothing. The engine and the inference subprocess
are replaced by fakes: no torch, no SpeechBrain, no network, no weights. Synthetic values only."""

import copy
import os
import tempfile
import unittest
from unittest import mock

from snitch_worker import engine_runtime, model_orders, runner, voice, voice_models, voice_separation
from snitch_worker.voice import VoiceProviderUnavailable

WORKER = "wk_synthetic_worker_identifier_0001"
REPO = "synthetic/sepformer-a"
REVISION = "5" * 40


def runtime(kind="separation", **changes):
    value = {"source": "on_demand", "family": "speechbrain", "engine": "speechbrain", "distribution": "speechbrain",
             "version": "1.0.3", "kind": kind, "kinds": ["fingerprint", "separation"], "python": "3.12", "platform": "linux-x86_64",
             "cuda": "", "disk_bytes": 1 << 20, "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 60}
    value.update(changes)
    return value


def order(**changes):
    value = {"order_id": "ord_synthetic_separation_order_0001", "worker_id": WORKER, "kind": "separation", "action": "install",
             "model_id": "sepformer-a", "engine": "speechbrain-sepformer", "repo": REPO, "revision": REVISION,
             "idempotency_key": "separation-install-0001", "status": "pending", "runtime": runtime(), "runtime_intent": "install"}
    value.update(changes)
    return value


class Response:
    def __init__(self, status_code=204, payload=None):
        self.status_code, self._payload = status_code, payload

    def json(self):
        return self._payload

    def raise_for_status(self):
        if self.status_code >= 400:
            raise RuntimeError("refused")


class Snitch:
    def __init__(self, orders=()):
        self.orders = [copy.deepcopy(o) for o in orders]
        self.acks, self.reports = [], []

    def get_httpx_client(self):
        return self

    def get(self, path, **_kwargs):
        return Response(200, {"worker_id": WORKER, "models": [], "remove": [], "orders": copy.deepcopy(self.orders)})

    def post(self, path, json=None, **_kwargs):
        if path.endswith("/ack"):
            self.acks.append(json)
        elif path == "/api/worker/voice-models/status":
            self.reports.append(json)
        else:
            return Response(404)
        return Response(204)

    def states(self):
        return [(r["state"], r["cause_code"]) for r in self.reports]


class Harness:
    def __init__(self, test):
        self.directory = tempfile.TemporaryDirectory()
        test.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models")}
        self.downloads, self.calls, self.engine_present = [], [], False
        self.proof = lambda: {}

        def download(repo, revision, target, progress):
            self.downloads.append((repo, revision))
            os.makedirs(target, exist_ok=True)
            with open(os.path.join(target, "masknet.ckpt"), "w") as handle:
                handle.write("synthetic")

        def unexpected(path, environ):
            raise AssertionError("a separation model is never proven by a diarization or a fingerprint")

        self.manager = voice_models.ModelManager(self.env, download, unexpected, None, min_progress_interval=0, verify_fingerprint=unexpected)
        self.executor = model_orders.OrderExecutor(self.manager, self.env, os.path.join(self.directory.name, ".secrets", "hf_token"))
        self.model = voice_models.model_path(self.env, REPO, REVISION)

    def patched(self):
        def install(rt, environ, report, detailed=False):
            self.calls.append(("install", rt["kind"]))
            self.engine_present = True

        def call(rt, path, environ, **request):
            self.calls.append((request["operation"], rt["kind"], path))
            return self.proof() if request["operation"] == "verify" else self.separation(request)

        return (mock.patch.object(engine_runtime, "install", install), mock.patch("snitch_worker.voice_runtime.call", call),
                mock.patch.object(engine_runtime, "installed", lambda rt, environ: self.engine_present))

    def sync(self, snitch):
        a, b, c = self.patched()
        with a, b, c:
            model_orders.sync_orders(snitch, self.executor)
            self.executor.wait()

    def separation(self, request):
        return {"windows": [{"sample_rate": 16000, "start": start or 0.0, "end": end or 2.0, "gain": 1.0,
                             "sources": [{"index": i, "path": os.path.join(request["output_dir"], f"window-{w}-source-{i}.wav"), "samples": 32000} for i in range(2)]}
                            for w, (start, end) in enumerate(request["windows"])]}

    def separate(self, **changes):
        arguments = {"repo": REPO, "revision": REVISION, "audio": "/clip.wav", "output_dir": os.path.join(self.directory.name, "out"),
                     "windows": [(None, None)]}
        arguments.update(changes)
        a, b, c = self.patched()
        with a, b, c:
            return voice_separation.separate(self.env, **arguments)


class InstallTests(unittest.TestCase):
    def test_nothing_is_downloaded_or_installed_without_an_order(self):
        h, snitch = Harness(self), Snitch()
        h.sync(snitch)
        self.assertEqual((h.downloads, h.calls, snitch.reports, snitch.acks), ([], [], [], []))

    def test_an_order_installs_the_model_and_it_is_ready_only_after_the_test_inference(self):
        h, snitch = Harness(self), Snitch([order()])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [{"accepted": True, "cause_code": ""}])
        self.assertEqual(h.downloads, [(REPO, REVISION)])
        self.assertEqual(h.calls, [("install", "separation"), ("verify", "separation", h.model)])
        states = [state for state, _ in snitch.states()]
        self.assertEqual(states[-2:], ["verifying", "ready"])
        self.assertLess(states.index("downloading"), states.index("verifying"))
        self.assertTrue(voice_models.is_ready(h.env, REPO, REVISION))
        self.assertEqual(engine_runtime.model_binding(h.model), runtime())

    def test_a_failed_test_inference_is_never_ready_and_carries_a_stable_cause(self):
        for error, cause in ((VoiceProviderUnavailable("inference_failed"), "inference_failed"),
                             (VoiceProviderUnavailable("model_load_failed"), "runtime_incompatible"),
                             (VoiceProviderUnavailable("gpu_out_of_memory"), "runtime_incompatible")):
            with self.subTest(cause=cause, reason=error.reason):
                h, snitch = Harness(self), Snitch([order()])

                def broken(error=error):
                    raise error

                h.proof = broken
                h.sync(snitch)
                self.assertEqual(snitch.states()[-1], ("failed", cause))
                self.assertNotIn("ready", [state for state, _ in snitch.states()])
                self.assertFalse(voice_models.is_ready(h.env, REPO, REVISION))
                self.assertIsNone(engine_runtime.model_binding(h.model))

    def test_an_order_without_the_engine_of_the_family_is_refused_at_reception(self):
        for bad in (order(runtime=None), order(runtime=runtime(kind="fingerprint")), order(engine="pyannote", runtime=None),
                    order(kind="separation", runtime=runtime(family="nemo", engine="nemo-toolkit", distribution="nemo-toolkit", kinds=["diarization", "fingerprint"]))):
            with self.subTest(bad=bad):
                h, snitch = Harness(self), Snitch([bad])
                h.sync(snitch)
                self.assertEqual(snitch.acks, [{"accepted": False, "cause_code": "invalid_order"}])
                self.assertEqual((h.downloads, h.calls, snitch.reports), ([], [], []))

    def test_a_disk_too_small_for_the_model_is_refused_before_any_download(self):
        h, snitch = Harness(self), Snitch([order(size_bytes=318_987_701)])
        h.executor._free_bytes = lambda path: 100_000_000
        h.sync(snitch)
        self.assertEqual(snitch.states(), [("failed", "disk_full")])
        self.assertEqual((h.downloads, h.calls), ([], []))

    def test_an_engine_that_does_not_fit_the_machine_fails_the_order_before_any_download(self):
        h, snitch = Harness(self), Snitch([order()])
        with mock.patch.object(engine_runtime, "install", side_effect=engine_runtime.RuntimeFailed("insufficient_ram")):
            model_orders.sync_orders(snitch, h.executor)
            h.executor.wait()
        self.assertEqual(snitch.states(), [("failed", "runtime_incompatible")])
        self.assertEqual(h.downloads, [])
        self.assertFalse(voice_models.is_ready(h.env, REPO, REVISION))

    def test_a_restart_repeats_the_outcome_without_a_second_download(self):
        h, snitch = Harness(self), Snitch([order()])
        h.sync(snitch)
        again = Snitch([order(status="acknowledged")])
        h.sync(again)
        self.assertEqual(len(h.downloads), 1)
        self.assertEqual(again.states(), [("ready", "")])


class SeparateTests(unittest.TestCase):
    """The entry point a caller uses (the benchmark step of #402): audio in, the sources of the model out."""

    def ready(self):
        h = Harness(self)
        h.sync(Snitch([order()]))
        h.calls.clear()
        return h

    def test_a_ready_model_separates_every_window_asked_in_one_call_of_its_own_environment(self):
        h = self.ready()
        results = h.separate(windows=[(1.5, 4.0), (6.0, 7.5), (None, None)])
        self.assertEqual(h.calls, [("infer", "separation", h.model)])  # one inference process for the three windows
        self.assertEqual([(r["start"], r["end"], len(r["sources"])) for r in results], [(1.5, 4.0, 2), (6.0, 7.5, 2), (0.0, 2.0, 2)])
        self.assertEqual(h.downloads, [(REPO, REVISION)])  # the order's download, nothing more

    def test_an_absent_model_is_an_explicit_state_and_downloads_nothing(self):
        h = Harness(self)
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            h.separate()
        self.assertEqual(caught.exception.reason, "model_path_missing")
        self.assertEqual((h.downloads, h.calls), ([], []))

    def test_a_downloaded_model_that_was_never_proven_is_not_ready(self):
        h = Harness(self)
        h.proof = mock.Mock(side_effect=VoiceProviderUnavailable("inference_failed"))
        h.sync(Snitch([order()]))
        h.calls.clear()
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            h.separate()
        self.assertEqual(caught.exception.reason, "model_path_missing")
        self.assertEqual(h.calls, [])

    def test_a_ready_model_whose_engine_disappeared_is_an_explicit_state_and_installs_nothing(self):
        h = self.ready()
        h.engine_present = False
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            h.separate()
        self.assertEqual(caught.exception.reason, "ml_libraries_missing")
        self.assertEqual(h.calls, [])

    def test_a_model_of_another_kind_is_never_used_to_separate(self):
        h = self.ready()
        engine_runtime.bind(h.model, runtime(kind="fingerprint"))
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            h.separate()
        self.assertEqual(caught.exception.reason, "ml_libraries_missing")
        self.assertEqual(h.calls, [])

    def test_an_answer_that_is_not_the_sources_of_the_output_directory_fails(self):
        h = self.ready()
        good = h.separation({"output_dir": os.path.join(h.directory.name, "out"), "windows": [(None, None)]})
        for answer in ({"windows": [{"sample_rate": 16000, "sources": []}]}, {"windows": [{"sources": [{"index": 0, "path": "/etc/passwd", "samples": 1}], "sample_rate": 16000}]},
                       {"windows": [{"sample_rate": "16000", "sources": [{"index": 0, "path": os.path.join(h.directory.name, "out", "source-0.wav"), "samples": 1}]}]},
                       {"windows": good["windows"] * 2}, {"windows": []}, {"sample_rate": 16000, "sources": good["windows"][0]["sources"]}):
            with self.subTest(answer=answer):
                h.separation = lambda request, answer=answer: answer
                with self.assertRaises(VoiceProviderUnavailable) as caught:
                    h.separate()
                self.assertEqual(caught.exception.reason, "inference_failed")

    def test_an_invalid_reference_or_window_is_refused_before_anything_runs(self):
        h = self.ready()
        for changes in ({"repo": "../escape"}, {"revision": "main"}, {"windows": [(0.0, 1.0), (3.0, 1.0)]}, {"windows": [(-1.0, None)]},
                        {"windows": [(float("nan"), None)]}, {"windows": []}, {"windows": [(1.0,)]}, {"windows": (0.0, 1.0)}):
            with self.subTest(changes=changes), self.assertRaises(VoiceProviderUnavailable):
                h.separate(**changes)
        self.assertEqual(h.calls, [])


class ProductionTests(unittest.TestCase):
    """Being ready opens no production job: the analysis of a clip has no separation step on the Worker."""

    def test_the_worker_has_no_separation_step(self):
        self.assertFalse(voice.is_voice_step("voice.separation.0"))
        self.assertFalse([step for step in runner.supported_steps() if "separation" in step])


if __name__ == "__main__":
    unittest.main()
