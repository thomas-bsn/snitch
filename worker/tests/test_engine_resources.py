"""Offline tests of the resource checks (#354): hardware report, refusals before install, device at inference."""
import tempfile
import unittest
from unittest.mock import Mock, patch

from snitch_worker import engine_runtime as runtime, model_orders, voice_runtime
from snitch_worker.voice import VoiceProviderUnavailable

GIB = 1 << 30


def manifest(**change):
    value = dict(schema=1, engine="nemo-titanet", version="1.0", adapter_version=1,
                 python="3.12", platform="linux-x86_64", cuda="12.4", license="Apache-2.0",
                 kind="fingerprint", architecture="titanet", disk_bytes=4096, vram_bytes=4 * GIB,
                 wheels=[dict(name="adapter-1-py3-none-any.whl", url="https://files.pythonhosted.org/packages/a.whl", sha256="a" * 64, size=100)])
    return value | change


class ManifestTests(unittest.TestCase):
    def test_legacy_manifest_stays_valid(self):
        runtime.validate(manifest(), "nemo-titanet", "fingerprint")

    def test_optional_constraints_and_contracts_are_accepted(self):
        value = manifest(ram_bytes=8 * GIB, capabilities=["fingerprint"], input_contract="audio-16k-mono", output_contract="embedding-v1", cpu_fallback=True)
        runtime.validate(value, "nemo-titanet", "fingerprint")

    def test_invalid_optionals_and_separation_are_refused(self):
        for change in [dict(ram_bytes=-1), dict(ram_bytes=True), dict(capabilities=["separation"]), dict(capabilities=["diarization"]),
                       dict(capabilities=["fingerprint", "fingerprint"]), dict(input_contract="Has Space"), dict(cpu_fallback="yes"), dict(unknown=1)]:
            with self.subTest(change=change):
                with self.assertRaises(runtime.RuntimeFailed):
                    runtime.validate(manifest(**change), "nemo-titanet", "fingerprint")


class CheckResourcesTests(unittest.TestCase):
    hw = dict(python="3.12", platform="linux-x86_64", cuda="12.4", ram_bytes=16 * GIB, vram_bytes=8 * GIB, disk_free_bytes=100 * GIB, cpu_fallback=False)

    def refused(self, value, hw):
        with self.assertRaises(runtime.RuntimeFailed) as raised:
            runtime.check_resources(value, hw)
        return raised.exception.cause

    def test_unknown_hardware_never_refuses(self):
        runtime.check_resources(manifest(ram_bytes=999 * GIB), None)

    def test_ram_is_refused_with_a_stable_cause(self):
        self.assertEqual(self.refused(manifest(ram_bytes=32 * GIB), self.hw), "insufficient_ram")
        runtime.check_resources(manifest(ram_bytes=8 * GIB), self.hw)

    def test_cuda_release_without_gpu_requires_a_distinct_cpu_release(self):
        no_gpu = self.hw | {"cuda": "", "vram_bytes": 0}
        self.assertEqual(self.refused(manifest(), no_gpu), "cuda_incompatible")
        self.assertEqual(self.refused(manifest(cpu_fallback=True), no_gpu), "cuda_incompatible")  # configuration missing
        self.assertEqual(self.refused(manifest(), no_gpu | {"cpu_fallback": True}), "cuda_incompatible")  # contract missing
        self.assertEqual(self.refused(manifest(cpu_fallback=True), no_gpu | {"cpu_fallback": True}), "cuda_incompatible")

    def test_cpu_release_needs_no_gpu(self):
        runtime.check_resources(manifest(cuda="cpu", vram_bytes=0), self.hw | {"cuda": "", "vram_bytes": 0})


class HardwareTests(unittest.TestCase):
    def test_hardware_reports_the_machine_and_the_configuration(self):
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory, runtime.CPU_FALLBACK_ENV: "1"}
            hw = runtime.hardware(env, gpu=lambda: ("12.4", 8 * GIB, 6 * GIB))
        self.assertEqual((hw["cuda"], hw["vram_bytes"], hw["cpu_fallback"]), ("12.4", 8 * GIB, True))
        self.assertRegex(hw["python"], r"^\d+\.\d+$")
        self.assertGreater(hw["disk_free_bytes"], 0)

    def test_no_gpu_means_no_cuda_and_no_vram(self):
        with tempfile.TemporaryDirectory() as directory:
            hw = runtime.hardware({"SNITCH_WORKER_MODELS_DIR": directory}, gpu=lambda: ("", 0, 0))
        self.assertEqual((hw["cuda"], hw["vram_bytes"], hw["cpu_fallback"]), ("", 0, False))

    def test_unassessable_gpu_reports_nothing(self):
        with tempfile.TemporaryDirectory() as directory:
            self.assertIsNone(runtime.hardware({"SNITCH_WORKER_MODELS_DIR": directory}, gpu=lambda: None))

    def test_executor_posts_hardware_once_until_it_changes(self):
        from snitch_worker import voice_models
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            executor = model_orders.OrderExecutor(voice_models.ModelManager(env, Mock(), Mock()), env, "")
            http = Mock()
            http.post.return_value.status_code = 204
            with patch.object(runtime, "gpu_state", return_value=("12.4", 8 * GIB, 8 * GIB)):
                executor.report_hardware(http)
                executor.report_hardware(http)
            self.assertEqual(http.post.call_count, 1)
            self.assertEqual(http.post.call_args.args[0], "/api/worker/voice-engines/hardware")
            self.assertNotIn("token", str(http.post.call_args.kwargs).lower())


class InstallRefusalTests(unittest.TestCase):
    def install(self, value, hw):
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            with patch.object(runtime.sys, "version_info", (3, 12)), patch.object(runtime.platform, "system", return_value="Linux"), \
                    patch.object(runtime.platform, "machine", return_value="x86_64"), patch.object(runtime, "hardware", return_value=hw), \
                    patch.object(runtime, "_download") as download:
                with self.assertRaises(runtime.RuntimeFailed) as raised:
                    runtime.install(value, env, lambda _: None)
                download.assert_not_called()  # refused before any download
                return raised.exception.cause

    def test_incompatible_platform_has_its_own_cause(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(runtime.platform, "system", return_value="Darwin"), patch.object(runtime, "_download") as download:
                with self.assertRaises(runtime.RuntimeFailed) as raised:
                    runtime.install(manifest(), {"SNITCH_WORKER_MODELS_DIR": directory}, lambda _: None)
                download.assert_not_called()
        self.assertEqual(raised.exception.cause, "platform_incompatible")

    def test_ram_and_cuda_refusals_never_download(self):
        hw = dict(python="3.12", platform="linux-x86_64", cuda="12.4", ram_bytes=4 * GIB, vram_bytes=8 * GIB, disk_free_bytes=100 * GIB, cpu_fallback=False)
        self.assertEqual(self.install(manifest(ram_bytes=8 * GIB), hw), "insufficient_ram")
        self.assertEqual(self.install(manifest(), hw | {"cuda": ""}), "cuda_incompatible")

    def test_vram_probe_exit_is_a_stable_cause(self):
        hw = dict(python="3.12", platform="linux-x86_64", cuda="12.4", ram_bytes=64 * GIB, vram_bytes=8 * GIB, disk_free_bytes=100 * GIB, cpu_fallback=False)
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}

            def run(command, **kwargs):
                return Mock(returncode=3 if command[2:3] == ["-c"] and "total_memory" in command[3] else 0)
            with patch.object(runtime.sys, "version_info", (3, 12)), patch.object(runtime.platform, "system", return_value="Linux"), \
                    patch.object(runtime.platform, "machine", return_value="x86_64"), patch.object(runtime, "hardware", return_value=hw), \
                    patch.object(runtime, "_download"), patch.object(runtime, "_wheel_budget"), patch.object(runtime.subprocess, "run", side_effect=run), \
                    patch.object(runtime.shutil, "disk_usage", return_value=Mock(free=100 * GIB)):
                with self.assertRaises(runtime.RuntimeFailed) as raised:
                    runtime.install(manifest(), env, lambda _: None)
            self.assertEqual(raised.exception.cause, "insufficient_vram")
            self.assertFalse(runtime.installed(manifest(), env))

    def test_engine_order_reports_resource_causes_unchanged(self):
        self.assertTrue({"insufficient_ram", "insufficient_vram", "cuda_incompatible", "platform_incompatible"} <= model_orders.ENGINE_CAUSES)


class InferenceDeviceTests(unittest.TestCase):
    def device(self, value, state, env=None):
        return voice_runtime.inference_device(value, env or {}, gpu=lambda: state)

    def test_cpu_release_runs_on_the_cpu(self):
        self.assertEqual(self.device(manifest(cuda="cpu", vram_bytes=0), ("", 0, 0)), "cpu")

    def test_cuda_release_uses_the_gpu_when_vram_is_free(self):
        self.assertEqual(self.device(manifest(), ("12.4", 8 * GIB, 6 * GIB)), "cuda")

    def test_short_free_vram_fails_with_a_stable_cause(self):
        with self.assertRaises(VoiceProviderUnavailable) as raised:
            self.device(manifest(), ("12.4", 8 * GIB, 1 * GIB))
        self.assertEqual(raised.exception.reason, "gpu_out_of_memory")

    def test_no_gpu_is_never_an_automatic_cpu_fallback(self):
        for value, env in [(manifest(), {}), (manifest(cpu_fallback=True), {}), (manifest(), {runtime.CPU_FALLBACK_ENV: "1"})]:
            with self.subTest(value=value.get("cpu_fallback"), env=env):
                with self.assertRaises(VoiceProviderUnavailable) as raised:
                    self.device(value, ("", 0, 0), env)
                self.assertEqual(raised.exception.reason, "cuda_unavailable")

    def test_cuda_release_cannot_become_a_cpu_release_through_configuration(self):
        with self.assertRaises(VoiceProviderUnavailable) as raised:
            self.device(manifest(cpu_fallback=True), ("", 0, 0), {runtime.CPU_FALLBACK_ENV: "1"})
        self.assertEqual(raised.exception.reason, "cuda_unavailable")

    def test_other_cuda_version_is_refused(self):
        with self.assertRaises(VoiceProviderUnavailable):
            self.device(manifest(), ("11.8", 8 * GIB, 8 * GIB))

    def test_unassessable_gpu_keeps_the_configured_device(self):
        self.assertIsNone(self.device(manifest(), None))

    def test_call_passes_the_decided_device_and_installs_nothing(self):
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory, "SNITCH_VOICE_DEVICE": "auto"}
            with patch.object(runtime, "installed", return_value=True), patch.object(runtime, "gpu_state", return_value=("", 0, 0)), \
                    patch.object(runtime, "install") as install, patch.object(voice_runtime.cancellation, "run_process") as run:
                with self.assertRaises(VoiceProviderUnavailable) as raised:
                    voice_runtime.call(manifest(), directory, env, operation="verify")
                self.assertEqual(raised.exception.reason, "cuda_unavailable")
                run.assert_not_called()
                install.assert_not_called()


if __name__ == "__main__":
    unittest.main()
