"""Updating the runtime of a Ready model and going back (#352). Synthetic values only, no network, no weights."""

import copy
import os
import tempfile
import unittest
from unittest import mock

from snitch_worker import engine_runtime, model_orders, voice_models
from snitch_worker.voice import VoiceProviderUnavailable

WORKER = "wk_synthetic_worker_identifier_0001"
REPO = "synthetic/fingerprint-a"
REVISION = "a" * 40


def manifest(version):
    return dict(schema=1, engine="nemo-titanet", version=version, adapter_version=1,
                python="3.12", platform="linux-x86_64", cuda="cpu", license="Apache-2.0",
                kind="fingerprint", architecture="titanet", disk_bytes=4096, vram_bytes=0,
                wheels=[dict(name="adapter-1-py3-none-any.whl", url="https://files.pythonhosted.org/packages/a.whl", sha256="a" * 64, size=100)])


def order(intent, version, number=1, **changes):
    value = {
        "order_id": f"ord_synthetic_runtime_order_{number:04d}",
        "worker_id": WORKER,
        "kind": "fingerprint",
        "action": "install",
        "model_id": "titanet-a",
        "engine": "nemo-titanet",
        "repo": REPO,
        "revision": REVISION,
        "idempotency_key": f"runtime-{intent}-{number:04d}",
        "status": "pending",
        "runtime": manifest(version),
        "runtime_intent": intent,
    }
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

    def last(self):
        return (self.reports[-1]["state"], self.reports[-1]["cause_code"])


class Harness:
    """A Worker volume that already holds the model, bound to runtime 1.0.0."""

    def __init__(self, test, directory=None):
        self.directory = directory or tempfile.TemporaryDirectory()
        if directory is None:
            test.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models")}
        self.downloads, self.calls = [], []
        self.manager = voice_models.ModelManager(self.env, lambda *a: self.downloads.append(a), lambda *a: None, None, min_progress_interval=0)
        self.executor = model_orders.OrderExecutor(self.manager, self.env, os.path.join(self.directory.name, ".secrets", "hf_token"))
        self.model = voice_models.model_path(self.env, REPO, REVISION)
        if directory is None:
            os.makedirs(self.model)
            with open(os.path.join(self.model, voice_models.READY_MARKER), "w") as handle:
                handle.write(REVISION)
            engine_runtime.bind(self.model, manifest("1.0.0"))
        self.verify = lambda runtime: None
        self.present = set()

    def binding(self):
        return engine_runtime.model_binding(self.model)

    def sync(self, snitch):
        def install(runtime, environ, report, detailed=False):
            self.calls.append(("install", runtime["version"]))
            for stage in ("downloading", "verifying"):
                report(stage)

        def call(runtime, path, environ, **request):
            # The binding is still the previous one while the new runtime is being proven.
            self.calls.append(("verify", runtime["version"], self.binding()["version"]))
            self.verify(runtime)
            return {}

        with mock.patch.object(engine_runtime, "install", install), \
                mock.patch("snitch_worker.voice_runtime.call", call), \
                mock.patch.object(engine_runtime, "installed", lambda runtime, environ: runtime["version"] in self.present | {"1.0.0"}):
            model_orders.sync_orders(snitch, self.executor)
            self.executor.wait()


class RuntimeUpdateTests(unittest.TestCase):
    def test_an_update_proves_the_new_runtime_on_the_existing_weights_then_binds_it(self):
        h = Harness(self)
        snitch = Snitch([order("update", "2.0.0")])
        h.sync(snitch)
        self.assertEqual(h.downloads, [])
        self.assertEqual(h.calls, [("install", "2.0.0"), ("verify", "2.0.0", "1.0.0")])
        self.assertEqual(snitch.last(), ("ready", ""))
        self.assertEqual(h.binding()["version"], "2.0.0")
        self.assertTrue(voice_models.is_ready(h.env, REPO, REVISION))

    def test_a_failed_proof_keeps_the_previous_binding(self):
        for error, cause in [(VoiceProviderUnavailable("inference_failed"), "inference_failed"), (VoiceProviderUnavailable("model_load_failed"), "runtime_incompatible")]:
            with self.subTest(cause):
                h = Harness(self)

                def broken(runtime, error=error):
                    raise error

                h.verify = broken
                snitch = Snitch([order("update", "2.0.0")])
                h.sync(snitch)
                self.assertEqual(snitch.last(), ("failed", cause))
                self.assertEqual(h.binding()["version"], "1.0.0")
                self.assertTrue(voice_models.is_ready(h.env, REPO, REVISION))
                self.assertEqual(h.downloads, [])

    def test_a_failed_installation_keeps_the_previous_binding(self):
        h = Harness(self)
        with mock.patch.object(engine_runtime, "install", side_effect=engine_runtime.RuntimeFailed("disk_full")):
            model_orders.sync_orders(Snitch([order("update", "2.0.0")]), h.executor)
            h.executor.wait()
        self.assertEqual(h.binding()["version"], "1.0.0")

    def test_an_update_never_downloads_missing_weights(self):
        h = Harness(self)
        os.remove(os.path.join(h.model, voice_models.READY_MARKER))
        snitch = Snitch([order("update", "2.0.0")])
        h.sync(snitch)
        self.assertEqual((h.downloads, h.calls), ([], []))
        self.assertEqual(snitch.last(), ("failed", "runtime_incompatible"))
        self.assertEqual(h.binding()["version"], "1.0.0")

    def test_a_restart_between_proof_and_publication_publishes_once_without_download(self):
        h = Harness(self)

        def die(runtime):
            raise SystemExit  # the process dies after proving the runtime, before the binding is written

        h.verify = die
        with self.assertRaises(SystemExit), mock.patch.object(engine_runtime, "install", lambda *a, **k: None), \
                mock.patch("snitch_worker.voice_runtime.call", lambda *a, **k: die(None)):
            h.executor._run(order("update", "2.0.0"), Snitch())
        self.assertEqual(h.binding()["version"], "1.0.0")
        again = Harness(self, directory=h.directory)
        snitch = Snitch([order("update", "2.0.0", status="acknowledged")])
        again.sync(snitch)
        again.sync(snitch)
        self.assertEqual(again.binding()["version"], "2.0.0")
        self.assertEqual([c[0] for c in again.calls].count("verify"), 1)
        self.assertEqual((h.downloads, again.downloads), ([], []))

    def test_a_redelivered_update_is_not_executed_twice(self):
        h = Harness(self)
        snitch = Snitch([order("update", "2.0.0")])
        h.sync(snitch)
        h.sync(snitch)
        self.assertEqual(h.calls.count(("install", "2.0.0")), 1)

    def test_a_key_reused_for_another_intent_is_refused(self):
        h = Harness(self)
        h.sync(Snitch([order("update", "2.0.0")]))
        snitch = Snitch([order("rollback", "1.0.0", number=2, idempotency_key="runtime-update-0001")])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [{"accepted": False, "cause_code": "invalid_order"}])

    def test_an_update_or_rollback_without_a_runtime_is_refused(self):
        for intent in ("update", "rollback", "other"):
            with self.subTest(intent):
                h = Harness(self)
                snitch = Snitch([order(intent, "2.0.0", runtime=None)])
                h.sync(snitch)
                self.assertEqual((snitch.acks, h.calls), ([{"accepted": False, "cause_code": "invalid_order"}], []))


class RollbackTests(unittest.TestCase):
    def bound_to_two(self):
        h = Harness(self)
        engine_runtime.bind(h.model, manifest("2.0.0"))
        return h

    def test_a_rollback_reuses_the_installed_version_without_installing(self):
        h = self.bound_to_two()
        snitch = Snitch([order("rollback", "1.0.0")])
        h.sync(snitch)
        self.assertEqual(h.calls, [("verify", "1.0.0", "2.0.0")])
        self.assertEqual(snitch.last(), ("ready", ""))
        self.assertEqual(h.binding()["version"], "1.0.0")
        self.assertEqual(h.downloads, [])

    def test_a_rollback_to_a_version_that_is_gone_fetches_nothing(self):
        h = self.bound_to_two()
        snitch = Snitch([order("rollback", "0.9.0")])
        h.sync(snitch)
        self.assertEqual(h.calls, [])
        self.assertEqual(snitch.last(), ("failed", "runtime_incompatible"))
        self.assertEqual(h.binding()["version"], "2.0.0")
        self.assertEqual(h.downloads, [])

    def test_a_failed_rollback_proof_keeps_the_current_binding(self):
        h = self.bound_to_two()

        def broken(runtime):
            raise VoiceProviderUnavailable("inference_failed")

        h.verify = broken
        snitch = Snitch([order("rollback", "1.0.0")])
        h.sync(snitch)
        self.assertEqual(snitch.last(), ("failed", "inference_failed"))
        self.assertEqual(h.binding()["version"], "2.0.0")


def on_demand(version):
    """The runtime of a NeMo family model (#376): the same family in two versions, each its own environment."""
    return {"source": "on_demand", "family": "nemo", "engine": "nemo-toolkit", "distribution": "nemo-toolkit",
            "version": version, "kind": "fingerprint", "kinds": ["diarization", "fingerprint"],
            "python": "3.12", "platform": "linux-x86_64", "cuda": "", "disk_bytes": 1 << 20,
            "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 600}


class OnDemandVersionTests(unittest.TestCase):
    """Update and rollback between two versions of an on-demand family reuse the weights and never download."""

    def bound(self, version="3.0.0", present=("3.0.0",)):
        h = Harness(self)
        engine_runtime.bind(h.model, on_demand(version))
        h.present = set(present)
        return h

    def test_an_update_proves_the_new_version_on_the_existing_weights_before_binding_it(self):
        h = self.bound()
        snitch = Snitch([order("update", "3.1.0", runtime=on_demand("3.1.0"))])
        h.sync(snitch)
        self.assertEqual(h.downloads, [])
        self.assertEqual(h.calls, [("install", "3.1.0"), ("verify", "3.1.0", "3.0.0")])
        self.assertEqual(snitch.last(), ("ready", ""))
        self.assertEqual(h.binding(), on_demand("3.1.0"))

    def test_a_failed_proof_keeps_the_binding_on_the_old_version(self):
        h = self.bound()

        def broken(runtime):
            raise VoiceProviderUnavailable("inference_failed")

        h.verify = broken
        snitch = Snitch([order("update", "3.1.0", runtime=on_demand("3.1.0"))])
        h.sync(snitch)
        self.assertEqual(snitch.last(), ("failed", "inference_failed"))
        self.assertEqual(h.binding(), on_demand("3.0.0"))
        self.assertEqual(h.downloads, [])

    def test_a_rollback_to_an_installed_version_installs_nothing(self):
        h = self.bound("3.1.0", present=("3.0.0", "3.1.0"))
        snitch = Snitch([order("rollback", "3.0.0", runtime=on_demand("3.0.0"))])
        h.sync(snitch)
        self.assertEqual(h.calls, [("verify", "3.0.0", "3.1.0")])
        self.assertEqual(snitch.last(), ("ready", ""))
        self.assertEqual(h.binding(), on_demand("3.0.0"))
        self.assertEqual(h.downloads, [])

    def test_a_rollback_to_a_lost_version_fails_cleanly_and_fetches_nothing(self):
        h = self.bound("3.1.0", present=("3.1.0",))
        snitch = Snitch([order("rollback", "3.0.0", runtime=on_demand("3.0.0"))])
        h.sync(snitch)
        self.assertEqual(h.calls, [])
        self.assertEqual(snitch.last(), ("failed", "runtime_incompatible"))
        self.assertEqual(h.binding(), on_demand("3.1.0"))
        self.assertEqual(h.downloads, [])


if __name__ == "__main__":
    unittest.main()
