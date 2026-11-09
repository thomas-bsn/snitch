"""On-demand engines (#370, ADR 0056): the Worker resolves a library with pip into an isolated environment, records
`pip freeze`, and reports stable causes only. The package index is a local wheelhouse, never the network.
Synthetic values only."""

import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_runtime as runtime_module
from snitch_worker import model_orders

import wheelhouse
from test_engine_orders import DIGEST, Harness, Snitch, order as engine_order

WHEELS = None


def setUpModule():
    global WHEELS, _TMP
    _TMP = tempfile.TemporaryDirectory()
    WHEELS = wheelhouse.build_nemo_wheelhouse(Path(_TMP.name) / "wheelhouse")


def tearDownModule():
    _TMP.cleanup()


def runtime(version="3.0.0", **changes):
    value = {"source": "on_demand", "family": "nemo", "engine": "nemo-toolkit", "distribution": "nemo-toolkit",
             "version": version, "kind": "diarization", "kinds": ["diarization", "fingerprint"],
             "python": "3.12", "platform": "linux-x86_64", "cuda": "", "disk_bytes": 1 << 20,
             "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 600}
    value.update(changes)
    return value


def linux():
    """The Worker runs on Linux x86-64 with Python 3.12; the tests may run anywhere."""
    return [patch.object(runtime_module.sys, "version_info", (3, 12)), patch.object(runtime_module.platform, "system", return_value="Linux"),
            patch.object(runtime_module.platform, "machine", return_value="x86_64")]


class OnDemandCase(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.env = self.environ()
        for p in linux():
            p.start()
            self.addCleanup(p.stop)

    def environ(self, **extra):
        env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models"), "PATH": os.environ.get("PATH", ""),
               "HOME": self.directory.name, "SNITCH_ENGINE_NO_INDEX": "1", "SNITCH_ENGINE_FIND_LINKS": str(WHEELS)}
        env.update(extra)
        return env

    def install(self, rt, env=None, report=None):
        return runtime_module.install(rt, env or self.env, report or (lambda *_: None), True)

    def failure(self, rt, env=None):
        with self.assertRaises(runtime_module.RuntimeFailed) as caught:
            self.install(rt, env)
        return caught.exception.cause


class InstallTests(OnDemandCase):
    def test_speechbrain_is_installed_with_a_torchaudio_that_still_has_the_apis_it_imports(self):
        # SpeechBrain 1.0.3 calls torchaudio.list_audio_backends() at import and declares torchaudio unbounded: pip
        # took the newest one (2.9+), which no longer has it, and the install failed with runtime_incompatible.
        family = runtime_module.FAMILIES["speechbrain"]
        self.assertEqual(runtime_module.on_demand_requirements(family, "1.0.3"), ["speechbrain==1.0.3", "torch<2.9", "torchaudio<2.9"])
        self.assertEqual(runtime_module.on_demand_requirements(runtime_module.FAMILIES["nemo"], "3.0.0"), ["nemo-toolkit[asr]==3.0.0"])
        self.assertEqual(runtime_module.on_demand_requirements(runtime_module.FAMILIES["wespeaker"], "1.0.1"), ["wespeakerruntime==1.0.1"])

    def test_install_records_the_exact_packages_and_a_ready_marker_last(self):
        steps = []
        rt = runtime()
        target = self.install(rt, report=steps.append)
        freeze = (target / "pip-freeze.txt").read_text().splitlines()
        self.assertIn("nemo-toolkit==3.0.0", freeze)
        self.assertIn("tinydep==1.0", freeze)  # the extra resolved its dependency
        self.assertEqual(steps, ["installing", "verifying"])
        self.assertTrue(runtime_module.installed(rt, self.env))
        # Local identity: family, version and the hash of the recorded freeze, never the slot of the directory.
        identity = runtime_module.local_identity(rt, self.env)
        expected = hashlib.sha256(json.dumps({"family": "nemo", "version": "3.0.0", "freeze_sha256": hashlib.sha256((target / "pip-freeze.txt").read_bytes()).hexdigest()},
                                             sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        self.assertEqual(identity, expected)
        self.assertNotEqual(identity, target.name)
        self.assertEqual(runtime_module.inventory(self.env), [{"engine": "nemo-toolkit", "local_identity": identity}])
        # A second order for the same engine installs nothing more.
        with patch.object(runtime_module.subprocess, "run") as run:
            self.assertEqual(self.install(rt), target)
            run.assert_not_called()

    def test_versions_coexist_each_in_its_own_environment(self):
        a, b = self.install(runtime("3.0.0")), self.install(runtime("3.1.0"))
        self.assertNotEqual(a, b)
        self.assertEqual(len(runtime_module.inventory(self.env)), 2)
        runtime_module.uninstall(runtime("3.0.0"), self.env)
        self.assertFalse(a.exists())
        self.assertEqual([i["local_identity"] for i in runtime_module.inventory(self.env)], [runtime_module.local_identity(runtime("3.1.0"), self.env)])
        runtime_module.uninstall(runtime("3.0.0"), self.env)  # idempotent

    def test_the_second_kind_reuses_the_environment_and_keeps_its_identity(self):
        diarization, fingerprint = runtime(), runtime(kind="fingerprint")
        target = self.install(diarization)
        identity = runtime_module.local_identity(diarization, self.env)
        with patch.object(runtime_module.subprocess, "run") as run:
            self.assertEqual(self.install(fingerprint), target)
            run.assert_not_called()
        self.assertEqual(runtime_module.local_identity(fingerprint, self.env), identity)
        self.assertTrue(runtime_module.installed(fingerprint, self.env))
        self.assertFalse(runtime_module.cleanup(fingerprint, self.env))  # a ready shared directory is never an incomplete one
        self.assertTrue(runtime_module.installed(diarization, self.env))

    def test_a_shared_directory_is_kept_while_a_model_of_another_kind_is_bound(self):
        diarization, fingerprint = runtime(), runtime(kind="fingerprint")
        target = self.install(diarization)
        model = Path(runtime_module.voice_models.models_root(self.env)) / "titanet"
        model.mkdir(parents=True)
        runtime_module.bind(str(model), fingerprint)
        with self.assertRaises(runtime_module.RuntimeFailed) as caught:
            runtime_module.uninstall(diarization, self.env)
        self.assertEqual(caught.exception.cause, "runtime_in_use")
        self.assertTrue(target.exists())
        self.assertTrue(runtime_module.installed(fingerprint, self.env))
        # Another version of the family is not the same directory and stays removable.
        other = self.install(runtime("3.1.0"))
        runtime_module.uninstall(runtime("3.1.0"), self.env)
        self.assertFalse(other.exists())
        # Once the last model is unbound the directory goes, whichever kind is asked.
        (model / ".snitch-runtime.json").unlink()
        runtime_module.uninstall(fingerprint, self.env)
        self.assertFalse(target.exists())

    def test_a_tampered_freeze_is_no_longer_ready(self):
        rt = runtime()
        target = self.install(rt)
        (target / "pip-freeze.txt").write_text("evil==1\n")
        self.assertFalse(runtime_module.installed(rt, self.env))
        self.assertEqual(runtime_module.inventory(self.env), [])


class FailureTests(OnDemandCase):
    def test_pip_failures_map_to_distinct_stable_causes(self):
        cases = {"9.9.9": "package_not_found", "7.0.0": "package_not_found", "5.0.0": "resolution_failed", "6.0.0": "build_failed"}
        for version, cause in cases.items():
            with self.subTest(version=version):
                self.assertEqual(self.failure(runtime(version)), cause)
                self.assertFalse(runtime_module.installed(runtime(version), self.env))

    def test_an_unreachable_index_is_a_network_failure(self):
        env = self.environ(SNITCH_ENGINE_INDEX_URL="http://127.0.0.1:9/simple")
        env.pop("SNITCH_ENGINE_NO_INDEX")
        env.pop("SNITCH_ENGINE_FIND_LINKS")
        self.assertEqual(self.failure(runtime(), env), "hf_network")

    def test_a_full_disk_is_refused_before_pip_runs(self):
        with patch.object(runtime_module.shutil, "disk_usage", return_value=shutil._ntuple_diskusage(10, 10, 0)), patch.object(runtime_module.subprocess, "run") as run:
            self.assertEqual(self.failure(runtime()), "disk_full")
            run.assert_not_called()

    def test_hardware_is_checked_before_installing(self):
        with patch.object(runtime_module, "hardware", return_value={"ram_bytes": 1 << 10, "cuda": "", "vram_bytes": 0, "disk_free_bytes": 1 << 40}), patch.object(runtime_module.subprocess, "run") as run:
            self.assertEqual(self.failure(runtime()), "insufficient_ram")
            run.assert_not_called()
        with patch.object(runtime_module.platform, "system", return_value="Darwin"):
            self.assertEqual(self.failure(runtime()), "platform_incompatible")

    def test_pip_output_is_never_relayed(self):
        with self.assertRaises(runtime_module.RuntimeFailed) as caught:
            self.install(runtime("9.9.9"))
        self.assertEqual(str(caught.exception), "package_not_found")
        self.assertNotIn("nemo", repr(caught.exception.__cause__))

    def test_the_cause_set_includes_the_new_causes(self):
        for cause in ("package_not_found", "resolution_failed", "build_failed"):
            self.assertIn(cause, model_orders.ENGINE_CAUSES)


class IsolationTests(OnDemandCase):
    def test_pip_gets_no_token_hooks_or_foreign_configuration(self):
        seen = []
        real = subprocess.run

        def spy(args, **kwargs):
            seen.append((list(args), dict(kwargs.get("env") or {})))
            return real(args, **kwargs)

        env = self.environ(HF_TOKEN="secret", SNITCH_HF_TOKEN_FILE="/secret", PYTHONPATH="/evil", PIP_INDEX_URL="http://evil", LD_PRELOAD="/evil", PYTHONSTARTUP="/evil")
        with patch.object(runtime_module.subprocess, "run", side_effect=spy):
            self.install(runtime(), env)
        pip_calls = [(a, e) for a, e in seen if "pip" in a]
        self.assertTrue(pip_calls)
        for args, call_env in seen:
            for key in ("HF_TOKEN", "SNITCH_HF_TOKEN_FILE", "PYTHONPATH", "PIP_INDEX_URL", "LD_PRELOAD", "PYTHONSTARTUP"):
                self.assertNotIn(key, call_env)
            self.assertNotIn("secret", " ".join(args))
        install_call = next(a for a, _ in pip_calls if "install" in a)
        self.assertIn("nemo-toolkit[asr]==3.0.0", install_call)
        self.assertIn("--no-index", install_call)
        self.assertIn("-I", install_call)  # isolated interpreter: no site hooks
        self.assertTrue(all(not a.startswith("-") or a.startswith("--") or a == "-I" or a == "-m" for a in install_call))


class CancellationAndCleanupTests(OnDemandCase):
    def test_cancel_stops_at_a_safe_point_and_cleanup_removes_only_that_directory(self):
        rt = runtime()
        calls = {"n": 0}

        def stop():
            calls["n"] += 1
            return calls["n"] >= 3

        other = self.install(runtime("3.1.0"))
        with runtime_module.stop_when(stop):
            self.assertEqual(self.failure(rt), "cancelled")
        target = runtime_module.location(rt, self.env)
        self.assertFalse((target / ".ready").exists())
        self.assertFalse(runtime_module.installed(rt, self.env))
        self.assertTrue(runtime_module.cleanup(rt, self.env))
        self.assertFalse(target.exists())
        self.assertTrue(runtime_module.installed(runtime("3.1.0"), self.env) and other.exists())
        self.assertFalse(runtime_module.cleanup(runtime("3.1.0"), self.env))  # a validated runtime is never removed


class ValidationTests(unittest.TestCase):
    def test_only_the_closed_family_table_is_accepted(self):
        runtime_module.validate_runtime(runtime())
        for bad in (runtime(family="unknown"), runtime(distribution="requests"), runtime(engine="nemo"), runtime(version="--index-url=http://evil"),
                    runtime(version="1.0 ; rm"), runtime(kind="separation"), runtime(python="3.9"), runtime(source="signed"), {**runtime(), "extra": 1}):
            with self.subTest(bad=bad), self.assertRaises(runtime_module.RuntimeFailed):
                runtime_module.validate_runtime(bad)


class OrderTests(OnDemandCase):
    """The same order mechanism as a signed release: ack, steps, outcome recorded before it is told, no duplicate after a restart."""

    def order(self, **changes):
        rt = changes.pop("runtime", runtime())
        return engine_order(engine="nemo-toolkit", version=rt["version"], runtime=rt, digest=DIGEST, **changes)

    def harness(self, directory=None):
        h = Harness(self, directory, install=lambda rt, environ, report, detailed=False: runtime_module.install(rt, self.env, report, detailed))
        h.executor._env = self.env
        return h

    def test_an_on_demand_order_installs_reports_its_identity_and_survives_a_restart(self):
        installs = []
        real = runtime_module.install

        def counting(rt, environ, report, detailed=False):
            installs.append(rt["version"])
            return real(rt, environ, report, detailed)

        o = self.order()
        snitch = Snitch([o])
        with patch.object(runtime_module, "install", counting):
            h = self.harness()
            h.sync(snitch)
            self.assertEqual(snitch.acks, [(o["order_id"], {"accepted": True, "cause_code": ""})])
            self.assertEqual(snitch.states(), ["installing", "verifying", "ready"])
            self.assertEqual(snitch.reports[-1]["local_identity"], runtime_module.local_identity(o["runtime"], self.env))
            # Same order again (a pull after a restart): nothing runs, the recorded outcome is repeated.
            h2 = self.harness(h.directory)
            h2.sync(snitch)
        self.assertEqual(installs, ["3.0.0"])
        self.assertEqual(snitch.reports[-1]["state"], "ready")

    def test_a_pip_failure_is_reported_with_its_stable_cause(self):
        o = self.order(runtime=runtime("5.0.0"))
        snitch = Snitch([o])
        self.harness().sync(snitch)
        self.assertEqual(snitch.states()[-1], "failed")
        self.assertEqual(snitch.reports[-1]["cause_code"], "resolution_failed")

    def test_a_malformed_on_demand_order_is_refused(self):
        o = self.order(runtime=runtime(family="unknown"))
        snitch = Snitch([o])
        self.harness().sync(snitch)
        self.assertEqual(snitch.acks, [(o["order_id"], {"accepted": False, "cause_code": "invalid_order"})])
        self.assertEqual(snitch.reports, [])

    def test_an_uninstall_order_removes_the_environment(self):
        rt = runtime()
        self.install(rt)
        o = self.order(intent="uninstall", idempotency_key="engine-uninstall-0001")
        snitch = Snitch([o])
        self.harness().sync(snitch)
        self.assertEqual(snitch.reports[-1]["state"], "removed")
        self.assertFalse(runtime_module.location(rt, self.env).exists())


class ModelOrderRuntimeTest(unittest.TestCase):
    """A model order carries the on-demand engine it waits for (#372); its kind is the model's."""

    def model_order(self, **changes):
        from snitch_worker.model_orders import OrderExecutor
        order = {"action": "install", "kind": "diarization", "engine": "nemo", "model_id": "synthetic-nemo-diar",
                 "repo": "synthetic/nemo-diar", "revision": "0" * 40, "idempotency_key": "model-ondemand-0001",
                 "runtime": runtime(), "runtime_intent": "install"}
        order.update(changes)
        return OrderExecutor._well_formed(order)

    def test_a_model_order_accepts_an_on_demand_runtime_of_its_kind(self):
        self.assertTrue(self.model_order())

    def test_a_model_order_refuses_a_runtime_of_another_kind_or_an_unknown_family(self):
        self.assertFalse(self.model_order(runtime=runtime(kind="fingerprint")))
        self.assertFalse(self.model_order(runtime=runtime(family="unknown")))

    def test_a_model_keeps_the_on_demand_runtime_it_was_bound_to(self):
        with tempfile.TemporaryDirectory() as directory:
            runtime_module.bind(directory, runtime())
            self.assertEqual(runtime_module.model_binding(directory), runtime())

    def test_the_embedded_contract_is_unchanged(self):
        self.assertTrue(self.model_order(runtime=None))


if __name__ == "__main__":
    unittest.main()
