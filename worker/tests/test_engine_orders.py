"""The Worker installs an approved engine only on an explicit order from Snitch (#349). Synthetic values only."""

import copy
import os
import tempfile
import unittest

from snitch_worker import engine_runtime, model_orders, voice_models

WORKER = "wk_synthetic_worker_identifier_0001"
DIGEST = "d" * 64


def manifest(version="1.0.0"):
    return dict(schema=1, engine="nemo-titanet", version=version, adapter_version=1,
                python="3.12", platform="linux-x86_64", cuda="cpu", license="Apache-2.0",
                kind="fingerprint", architecture="titanet", disk_bytes=4096, vram_bytes=0,
                wheels=[dict(name="adapter-1-py3-none-any.whl", url="https://files.pythonhosted.org/packages/a.whl", sha256="a" * 64, size=100)])


ORDER = {
    "order_id": "ord_synthetic_engine_order_0001",
    "worker_id": WORKER,
    "digest": DIGEST,
    "engine": "nemo-titanet",
    "version": "1.0.0",
    "intent": "install",
    "generation": 1,
    "idempotency_key": "engine-install-0001",
    "status": "pending",
    "runtime": manifest(),
}


class Response:
    def __init__(self, status_code=204, payload=None):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload

    def raise_for_status(self):
        if self.status_code >= 400:
            raise RuntimeError("refused")


class Snitch:
    """A Snitch that delivers engine orders and records acknowledgements, reports and inventories."""

    def __init__(self, engine_orders=()):
        self.engine_orders = [copy.deepcopy(o) for o in engine_orders]
        self.acks, self.reports, self.inventories = [], [], []

    def get_httpx_client(self):
        return self

    def get(self, path, **_kwargs):
        assert path == "/api/worker/voice-models"
        return Response(200, {"worker_id": WORKER, "models": [], "remove": [], "orders": [], "engine_orders": copy.deepcopy(self.engine_orders)})

    def post(self, path, json=None, **_kwargs):
        if path.startswith("/api/worker/voice-engine-orders/") and path.endswith("/ack"):
            self.acks.append((path.split("/")[-2], json))
        elif path == "/api/worker/voice-engines/status":
            self.reports.append(json)
        elif path == "/api/worker/voice-engines/inventory":
            self.inventories.append(json)
        else:
            return Response(404)
        return Response(204)

    def states(self):
        return [r["state"] for r in self.reports]


class Harness:
    def __init__(self, test, directory=None, install=None):
        self.directory = directory or tempfile.TemporaryDirectory()
        if directory is None:
            test.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models")}
        self.installed = []

        def default_install(runtime, environ, report, detailed=False):
            self.installed.append(runtime)
            self.test_detailed = detailed
            for stage in ("downloading", "installing", "verifying"):
                report(stage)
            return engine_runtime.location(runtime, environ)

        manager = voice_models.ModelManager(self.env, lambda *a: None, lambda *a: None, None, min_progress_interval=0)
        self.executor = model_orders.OrderExecutor(manager, self.env, os.path.join(self.directory.name, ".secrets", "hf_token"))
        self.executor._install_runtime = install or default_install

    def sync(self, snitch):
        model_orders.sync_orders(snitch, self.executor)
        self.executor.wait()


def order(**changes):
    value = copy.deepcopy(ORDER)
    value.update(changes)
    return value


class EngineOrderTests(unittest.TestCase):
    def test_nothing_is_installed_without_an_order(self):
        h = Harness(self)
        snitch = Snitch()
        h.sync(snitch)
        h.sync(snitch)
        self.assertEqual((h.installed, snitch.reports, snitch.acks), ([], [], []))

    def test_an_order_is_acknowledged_installed_and_reported_step_by_step(self):
        h = Harness(self)
        snitch = Snitch([ORDER])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [(ORDER["order_id"], {"accepted": True, "cause_code": ""})])
        self.assertEqual(snitch.states(), ["downloading_runtime", "installing", "verifying", "ready"])
        self.assertTrue(h.test_detailed)
        self.assertEqual(h.installed, [manifest()])
        last = snitch.reports[-1]
        # The signed digest is echoed back; the local identity is the Worker's own canonical hash.
        self.assertEqual((last["digest"], last["order_id"], last["cause_code"]), (DIGEST, ORDER["order_id"], ""))
        self.assertEqual(last["local_identity"], engine_runtime.identity(manifest()))
        self.assertNotEqual(last["local_identity"], DIGEST)

    def test_a_redelivered_order_is_never_installed_twice_even_after_a_restart(self):
        h = Harness(self)
        snitch = Snitch([ORDER])
        h.sync(snitch)
        h.sync(snitch)
        self.assertEqual(len(h.installed), 1)
        # A new process on the same volume: Snitch lost the report, the Worker repeats it and installs nothing.
        again = Harness(self, directory=h.directory)
        replay = Snitch([order(status="acknowledged")])
        again.sync(replay)
        self.assertEqual(again.installed, [])
        self.assertEqual(replay.states(), ["ready"])

    def test_an_unfinished_order_resumes_after_a_restart_without_duplication(self):
        def crash(runtime, environ, report, detailed=False):
            raise SystemExit  # the process dies mid-install: no outcome is recorded

        h = Harness(self, install=crash)
        snitch = Snitch([ORDER])
        with self.assertRaises(SystemExit):
            h.executor._run_engine(dict(ORDER, target="engine"), snitch)
        again = Harness(self, directory=h.directory)
        replay = Snitch([order(status="acknowledged")])
        again.sync(replay)
        again.sync(replay)
        self.assertEqual(len(again.installed), 1)
        self.assertEqual(replay.states()[-1], "ready")

    def test_a_failure_is_reported_with_a_stable_cause(self):
        for raised, expected in [("disk_full", "disk_full"), ("hf_network", "hf_network"), ("surprise", "runtime_incompatible")]:
            with self.subTest(raised):
                def install(runtime, environ, report, detailed=False):
                    raise engine_runtime.RuntimeFailed(raised)

                h = Harness(self, install=install)
                snitch = Snitch([ORDER])
                h.sync(snitch)
                self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", expected))
                self.assertNotIn("ready", snitch.states())

    def test_an_unexpected_error_fails_the_order_instead_of_leaving_it_open(self):
        def install(runtime, environ, report, detailed=False):
            raise ValueError("boom")

        h = Harness(self, install=install)
        snitch = Snitch([ORDER])
        h.sync(snitch)
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "runtime_incompatible"))

    def test_a_malformed_or_foreign_order_is_refused_and_installs_nothing(self):
        for name, change in [
            ("other worker", dict(worker_id="wk_synthetic_other_identifier_00009")),
            ("unknown intent", dict(intent="remove")),
            ("bad digest", dict(digest="x")),
            ("engine mismatch", dict(engine="pyannote")),
            ("version mismatch", dict(version="9.9.9")),
            ("generation", dict(generation=0)),
            ("runtime", dict(runtime=manifest() | {"import_path": "evil"})),
        ]:
            with self.subTest(name):
                h = Harness(self)
                snitch = Snitch([order(**change)])
                h.sync(snitch)
                self.assertEqual(h.installed, [])
                self.assertEqual(snitch.acks, [(ORDER["order_id"], {"accepted": False, "cause_code": "invalid_order"})])

    def test_a_key_reused_with_other_content_is_refused(self):
        h = Harness(self)
        h.sync(Snitch([ORDER]))
        snitch = Snitch([order(order_id="ord_synthetic_engine_order_0002", version="2.0.0", runtime=manifest("2.0.0"))])
        h.sync(snitch)
        self.assertEqual(len(h.installed), 1)
        self.assertEqual(snitch.acks[0][1]["accepted"], False)


class InventoryTests(unittest.TestCase):
    def test_the_inventory_lists_proven_runtimes_and_orders_nothing(self):
        h = Harness(self)
        self.assertEqual(engine_runtime.inventory(h.env), [])
        ready = engine_runtime.location(manifest(), h.env)
        (ready / "bin").mkdir(parents=True)
        (ready / "bin/python").touch()
        (ready / ".ready").write_text(engine_runtime.identity(manifest()))
        partial = engine_runtime.location(manifest("2.0.0"), h.env)
        (partial / "bin").mkdir(parents=True)
        (partial / "bin/python").touch()  # no ready marker: not proven
        self.assertEqual(engine_runtime.inventory(h.env), [{"engine": "nemo-titanet", "local_identity": engine_runtime.identity(manifest())}])
        snitch = Snitch()
        h.sync(snitch)
        self.assertEqual(snitch.inventories, [{"runtimes": [{"engine": "nemo-titanet", "local_identity": engine_runtime.identity(manifest())}]}])
        # Reconciling again right away repeats nothing, and a pull never installs anything.
        h.sync(snitch)
        self.assertEqual((len(snitch.inventories), h.installed, snitch.reports), (1, [], []))

    def test_a_lost_runtime_is_reported_as_absent_without_any_download(self):
        h = Harness(self)
        snitch = Snitch()
        h.sync(snitch)
        self.assertEqual(snitch.inventories, [{"runtimes": []}])
        self.assertEqual(h.installed, [])

    def _present(self, h, version="1.0.0"):
        path = engine_runtime.location(manifest(version), h.env)
        (path / "bin").mkdir(parents=True)
        (path / "bin/python").touch()
        (path / ".ready").write_text(engine_runtime.identity(manifest(version)))
        return path

    def test_an_uninstall_removes_only_the_designated_release_and_reports_it(self):
        h = Harness(self)
        target, other = self._present(h), self._present(h, "2.0.0")
        snitch = Snitch([order(intent="uninstall", idempotency_key="engine-uninstall-001")])
        h.sync(snitch)
        self.assertEqual(snitch.states(), ["removed"])
        self.assertEqual((snitch.reports[0]["cause_code"], h.installed), ("", []))
        self.assertFalse(target.exists())
        self.assertTrue((other / ".ready").exists())

    def test_an_uninstall_is_idempotent_and_never_repeated_after_a_restart(self):
        h = Harness(self)
        self._present(h)
        snitch = Snitch([order(intent="uninstall", idempotency_key="engine-uninstall-001")])
        h.sync(snitch)
        restarted = Harness(self, h.directory)
        restarted.sync(snitch)
        self.assertEqual(snitch.states(), ["removed", "removed"])
        # An absent runtime is already removed: the same removal under another order succeeds without error.
        again = Snitch([order(order_id="ord_synthetic_engine_order_0002", intent="uninstall", generation=2, idempotency_key="engine-uninstall-002")])
        restarted.sync(again)
        self.assertEqual(again.states(), ["removed"])

    def test_a_runtime_a_model_is_still_bound_to_is_not_removed(self):
        h = Harness(self)
        target = self._present(h)
        model = os.path.join(h.directory.name, "models")
        bound = os.path.join(model, "catalog", "org__repo", "rev1")
        os.makedirs(bound)
        engine_runtime.bind(bound, manifest())
        snitch = Snitch([order(intent="uninstall", idempotency_key="engine-uninstall-001")])
        h.sync(snitch)
        self.assertEqual((snitch.states(), snitch.reports[0]["cause_code"]), (["failed"], "runtime_in_use"))
        self.assertTrue((target / ".ready").exists())
        self.assertTrue(os.path.exists(os.path.join(bound, ".snitch-runtime.json")))


if __name__ == "__main__":
    unittest.main()
