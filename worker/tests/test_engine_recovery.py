"""Leaving a failed engine installation (#351): cooperative cancellation and bounded cleanup. Synthetic values only."""

import copy
import os
import threading
import unittest

from snitch_worker import engine_runtime, model_orders

from test_engine_orders import DIGEST, ORDER, Harness, Response, Snitch, manifest, order


class RecoverySnitch(Snitch):
    """A Snitch that also delivers cleanups and records their confirmation."""

    def __init__(self, engine_orders=(), cleanups=()):
        super().__init__(engine_orders)
        self.cleanups = [copy.deepcopy(c) for c in cleanups]
        self.cleaned = []

    def get(self, path, **kwargs):
        response = super().get(path, **kwargs)
        response._payload["engine_cleanups"] = copy.deepcopy(self.cleanups)
        return response

    def post(self, path, json=None, **kwargs):
        if path == "/api/worker/voice-engines/cleanup":
            self.cleaned.append(json)
            return Response(204)
        return super().post(path, json=json, **kwargs)


def cleanup_request(runtime=None, order_id="ord_synthetic_engine_order_0001"):
    return {"order_id": order_id, "digest": DIGEST, "runtime": runtime or manifest()}


class CancellationTests(unittest.TestCase):
    def test_a_running_install_stops_at_its_next_safe_point_and_confirms(self):
        started, release = threading.Event(), threading.Event()
        steps = []

        def install(runtime, environ, report, detailed=False):
            report("downloading")
            started.set()
            release.wait(5)
            steps.append("before-checkpoint")
            engine_runtime.checkpoint()  # the safe point inside the real install
            steps.append("after-checkpoint")
            return engine_runtime.location(runtime, environ)

        h = Harness(self, install=install)
        snitch = RecoverySnitch([ORDER])
        h.executor.handle(snitch.get("/api/worker/voice-models").json(), snitch)
        self.assertTrue(started.wait(5))
        # Snitch asks again on the next pull, now with the cancellation flag.
        h.executor.handle(RecoverySnitch([order(status="acknowledged", cancel_requested=True)]).get("/api/worker/voice-models").json(), snitch)
        release.set()
        h.executor.wait()
        self.assertEqual(steps, ["before-checkpoint"])
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "cancelled"))
        self.assertNotIn("ready", snitch.states())
        # A restart repeats the confirmation without installing anything.
        again = Harness(self, directory=h.directory)
        replay = RecoverySnitch([order(status="acknowledged", cancel_requested=True)])
        again.sync(replay)
        self.assertEqual(again.installed, [])
        self.assertEqual([(r["state"], r["cause_code"]) for r in replay.reports], [("failed", "cancelled")])

    def test_an_order_that_never_started_is_cancelled_without_installing(self):
        h = Harness(self)
        snitch = RecoverySnitch([order(status="acknowledged", cancel_requested=True)])
        h.sync(snitch)
        self.assertEqual(h.installed, [])
        self.assertEqual([(r["state"], r["cause_code"]) for r in snitch.reports], [("failed", "cancelled")])
        # Pulling again repeats the confirmation, never a second install.
        h.sync(snitch)
        self.assertEqual(h.installed, [])
        self.assertEqual({(r["state"], r["cause_code"]) for r in snitch.reports}, {("failed", "cancelled")})

    def test_an_install_that_finished_first_stays_ready(self):
        h = Harness(self)
        h.sync(Snitch([ORDER]))
        snitch = RecoverySnitch([order(status="acknowledged", cancel_requested=True)])
        h.sync(snitch)
        self.assertEqual(len(h.installed), 1)
        self.assertEqual([r["state"] for r in snitch.reports], ["ready"])


class CleanupTests(unittest.TestCase):
    def incomplete(self, h, version="1.0.0"):
        target = engine_runtime.location(manifest(version), h.env)
        (target / "lib").mkdir(parents=True)
        (target / "lib/partial.whl").write_text("partial")
        return target

    def test_only_the_incomplete_files_of_the_release_are_removed(self):
        h = Harness(self)
        target = self.incomplete(h)
        other = engine_runtime.location(manifest("2.0.0"), h.env)
        (other / "bin").mkdir(parents=True)
        (other / "bin/python").touch()
        (other / ".ready").write_text(engine_runtime.identity(manifest("2.0.0")))
        snitch = RecoverySnitch(cleanups=[cleanup_request()])
        h.sync(snitch)
        self.assertFalse(target.exists())
        self.assertTrue((other / ".ready").exists())
        self.assertEqual(snitch.cleaned, [{"order_id": "ord_synthetic_engine_order_0001", "digest": DIGEST, "removed": True}])

    def test_a_validated_runtime_is_never_removed(self):
        h = Harness(self)
        ready = engine_runtime.location(manifest(), h.env)
        (ready / "bin").mkdir(parents=True)
        (ready / "bin/python").touch()
        (ready / ".ready").write_text(engine_runtime.identity(manifest()))
        snitch = RecoverySnitch(cleanups=[cleanup_request()])
        h.sync(snitch)
        self.assertTrue((ready / ".ready").exists())
        self.assertEqual([c["removed"] for c in snitch.cleaned], [False])

    def test_nothing_to_remove_is_confirmed_and_repeating_is_harmless(self):
        h = Harness(self)
        snitch = RecoverySnitch(cleanups=[cleanup_request()])
        h.sync(snitch)
        h.sync(snitch)
        self.assertEqual([c["removed"] for c in snitch.cleaned], [False, False])

    def test_model_files_and_other_volumes_are_out_of_reach(self):
        h = Harness(self)
        models = os.path.join(h.env.get("SNITCH_VOICE_MODELS_DIR", h.directory.name), "keep")
        os.makedirs(models, exist_ok=True)
        keep = os.path.join(models, "weights.bin")
        with open(keep, "w") as handle:
            handle.write("weights")
        snitch = RecoverySnitch(cleanups=[cleanup_request(runtime=manifest() | {"import_path": "evil"}), cleanup_request(order_id="bad")])
        h.sync(snitch)
        self.assertEqual(snitch.cleaned, [])
        self.assertTrue(os.path.exists(keep))

    def test_nothing_is_removed_while_an_order_runs(self):
        started, release = threading.Event(), threading.Event()

        def install(runtime, environ, report, detailed=False):
            started.set()
            release.wait(5)
            return engine_runtime.location(runtime, environ)

        h = Harness(self, install=install)
        target = self.incomplete(h)
        busy = RecoverySnitch([order(order_id="ord_synthetic_engine_order_0009", idempotency_key="engine-install-0009", version="2.0.0", runtime=manifest("2.0.0"))], [cleanup_request()])
        h.executor.handle(busy.get("/api/worker/voice-models").json(), busy)
        self.assertTrue(started.wait(5))
        h.executor.handle(busy.get("/api/worker/voice-models").json(), busy)
        self.assertTrue(target.exists())
        self.assertEqual(busy.cleaned, [])
        release.set()
        h.executor.wait()
        h.sync(RecoverySnitch(cleanups=[cleanup_request()]))
        self.assertFalse(target.exists())


if __name__ == "__main__":
    unittest.main()
