"""SpeechBrain family on demand (#374, ADR 0056): the engine installs for fingerprint and for separation, ECAPA-TDNN
runs through a generic adapter, separation has no adapter and stays disabled. The package index is the local
wheelhouse of #370; no torch or SpeechBrain is needed. Synthetic values only."""

import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_runtime as runtime_module
from snitch_worker import model_orders

import wheelhouse

WHEELS = None


def setUpModule():
    global WHEELS, _TMP
    _TMP = tempfile.TemporaryDirectory()
    WHEELS = wheelhouse.build_speechbrain_wheelhouse(Path(_TMP.name) / "wheelhouse")


def tearDownModule():
    _TMP.cleanup()


def runtime(version="1.0.3", kind="fingerprint", **changes):
    value = {"source": "on_demand", "family": "speechbrain", "engine": "speechbrain", "distribution": "speechbrain",
             "version": version, "kind": kind, "kinds": ["fingerprint", "separation"],
             "python": "3.12", "platform": "linux-x86_64", "cuda": "", "disk_bytes": 1 << 20,
             "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 600}
    value.update(changes)
    return value


class SpeechBrainCase(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models"), "PATH": os.environ.get("PATH", ""),
                    "HOME": self.directory.name, "SNITCH_ENGINE_NO_INDEX": "1", "SNITCH_ENGINE_FIND_LINKS": str(WHEELS)}
        for p in (patch.object(runtime_module.sys, "version_info", (3, 12)), patch.object(runtime_module.platform, "system", return_value="Linux"),
                  patch.object(runtime_module.platform, "machine", return_value="x86_64")):
            p.start()
            self.addCleanup(p.stop)

    def install(self, rt, env=None):
        return runtime_module.install(rt, env or self.env, lambda *_: None, True)


class InstallTests(SpeechBrainCase):
    def test_the_engine_installs_for_a_fingerprint_model(self):
        rt = runtime()
        target = self.install(rt)
        self.assertIn("speechbrain==1.0.3", (target / "pip-freeze.txt").read_text().splitlines())
        self.assertTrue(runtime_module.installed(rt, self.env))
        self.assertEqual(runtime_module.inventory(self.env), [{"engine": "speechbrain", "local_identity": runtime_module.local_identity(rt, self.env)}])

    def test_the_engine_installs_for_a_separation_model_without_activating_anything(self):
        rt = runtime(kind="separation")
        target = self.install(rt)
        self.assertTrue(runtime_module.installed(rt, self.env))
        # Same family and version: the one environment serves both kinds.
        self.assertEqual(self.install(runtime(kind="fingerprint")), target)
        self.assertEqual([p.name for p in target.parent.iterdir() if p.is_dir()], [target.name])

    def test_it_never_shares_a_directory_with_the_signed_ecapa_engine(self):
        self.install(runtime())
        self.assertFalse((Path(self.env["SNITCH_WORKER_MODELS_DIR"]).parent / "runtimes" / "speechbrain-ecapa").exists())

    def test_a_version_without_the_family_library_is_not_ready(self):
        with self.assertRaises(runtime_module.RuntimeFailed):
            self.install(runtime("2.0.0"))
        self.assertFalse(runtime_module.installed(runtime("2.0.0"), self.env))

    def test_pip_gets_the_family_package_and_no_token_hook_or_foreign_configuration(self):
        seen = []
        real = runtime_module.subprocess.run

        def spy(args, **kwargs):
            seen.append((list(args), dict(kwargs.get("env") or {})))
            return real(args, **kwargs)

        self.env.update(HF_TOKEN="secret", PYTHONPATH="/evil", PIP_INDEX_URL="http://evil")
        with patch.object(runtime_module.subprocess, "run", side_effect=spy):
            self.install(runtime())
        install_call = next(a for a, _ in seen if "pip" in a and "install" in a)
        self.assertIn("speechbrain==1.0.3", install_call)
        for args, env in seen:
            for key in ("HF_TOKEN", "PYTHONPATH", "PIP_INDEX_URL"):
                self.assertNotIn(key, env)


class RunTests(SpeechBrainCase):
    """The Worker runs an installed family through the fixed bridge, in the family's own environment."""

    def call(self, rt, **request):
        from snitch_worker import voice_runtime
        started = []

        def fake(command, **kwargs):
            started.append((command, kwargs["env"]))
            Path(command[-1]).write_text(json.dumps({"result": {"dimensions": 192}}))

        with patch.object(voice_runtime.cancellation, "run_process", side_effect=fake):
            return voice_runtime.call(rt, "/model", self.env, **request), started

    def test_a_fingerprint_call_runs_in_the_installed_environment_on_the_cpu_when_there_is_no_gpu(self):
        rt = runtime()
        target = self.install(rt)
        with patch.object(runtime_module, "gpu_state", return_value=(False, 0, 0)):
            result, started = self.call(rt, operation="infer")
        self.assertEqual(result, {"dimensions": 192})
        command, env = started[0]
        self.assertEqual(command[0], str(target / "bin/python"))
        self.assertEqual(env["SNITCH_VOICE_DEVICE"], "cpu")
        self.assertNotIn("HF_TOKEN", env)
        self.assertEqual(env["HF_HUB_OFFLINE"], "1")

    def test_a_gpu_is_used_when_it_has_room(self):
        self.install(runtime())
        with patch.object(runtime_module, "gpu_state", return_value=("12.8", 24 << 30, 20 << 30)):
            _, started = self.call(runtime(), operation="infer")
        self.assertEqual(started[0][1]["SNITCH_VOICE_DEVICE"], "cuda")

    def test_a_missing_environment_never_installs_anything(self):
        from snitch_worker import voice_runtime
        from snitch_worker.voice import VoiceProviderUnavailable
        with patch.object(runtime_module, "install") as install, self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_runtime.call(runtime(), "/model", self.env, operation="verify")
        self.assertEqual(caught.exception.reason, "ml_libraries_missing")
        install.assert_not_called()

    def test_a_model_binding_of_the_family_round_trips(self):
        with tempfile.TemporaryDirectory() as directory:
            runtime_module.bind(directory, runtime())
            self.assertEqual(runtime_module.model_binding(directory), runtime())


class ValidationTests(unittest.TestCase):
    def test_the_family_accepts_fingerprint_and_separation_only(self):
        runtime_module.validate_runtime(runtime())
        runtime_module.validate_runtime(runtime(kind="separation"))
        for bad in (runtime(kind="diarization"), runtime(kinds=["fingerprint"]), runtime(engine="speechbrain-ecapa"),
                    runtime(distribution="speechbrain-ecapa"), runtime(version="--extra-index-url=http://evil"), runtime(cuda="cpu")):
            with self.subTest(bad=bad), self.assertRaises(runtime_module.RuntimeFailed):
                runtime_module.validate_runtime(bad)

    def test_the_signed_ecapa_release_is_still_validated_as_signed(self):
        release = json.loads((Path(__file__).parent.parent / "engine-releases" / "speechbrain-ecapa-1.0.3-r1" / "manifest.json").read_text())
        runtime_module.validate(release, "speechbrain-ecapa", "fingerprint")
        runtime_module.validate_runtime(release)
        self.assertFalse(runtime_module.is_on_demand(release))

    def test_an_engine_order_for_the_family_is_well_formed(self):
        order = {"intent": "install", "engine": "speechbrain", "digest": "a" * 64, "version": "1.0.3", "generation": 1,
                 "idempotency_key": "engine-install-0001", "runtime": runtime()}
        self.assertTrue(model_orders.OrderExecutor._well_formed_engine(order))


if __name__ == "__main__":
    unittest.main()
