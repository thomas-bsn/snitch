"""Offline regression tests for the executable trust boundary."""
import hashlib
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

from snitch_worker import engine_runtime as runtime, voice_runtime
from snitch_worker.voice import VoiceProviderUnavailable


def manifest():
    return dict(schema=1, engine="nemo-titanet", version="1.0", adapter_version=1,
                python="3.12", platform="linux-x86_64", cuda="cpu", license="Apache-2.0",
                kind="fingerprint", architecture="titanet", disk_bytes=4096, vram_bytes=0,
                wheels=[dict(name="adapter-1-py3-none-any.whl", url="https://files.pythonhosted.org/packages/a.whl", sha256="a" * 64, size=100)])


class EngineRuntimeTests(unittest.TestCase):
    def test_validation_rejects_unapproved_code_and_contracts(self):
        for change in [dict(engine="evil"), dict(kind="separation"), dict(architecture="custom"),
                       dict(python="3.11"), dict(version="../escape"), dict(schema=True), dict(adapter_version=True),
                       dict(disk_bytes=True), dict(cuda="latest"), dict(vram_bytes=-1),
                       dict(license=""), dict(import_path="evil")]:
            with self.subTest(change=change):
                value = manifest() | change
                with self.assertRaises(runtime.RuntimeFailed):
                    runtime.validate(value, value["engine"], value["kind"])
        for change in [dict(url="https://evil.example/a.whl"), dict(name="../a.whl"),
                       dict(name="a.tar.gz"), dict(sha256="latest"), dict(size=True),
                       dict(url="https://files.pythonhosted.org/a.whl?token=secret")]:
            with self.subTest(change=change):
                value = manifest()
                value["wheels"][0].update(change)
                with self.assertRaises(runtime.RuntimeFailed):
                    runtime.validate(value, value["engine"], value["kind"])

    def test_binding_atomic_and_version_specific(self):
        with tempfile.TemporaryDirectory() as directory:
            self.assertIsNone(runtime.model_binding(directory))
            runtime.bind(directory, manifest())
            self.assertEqual(runtime.model_binding(directory), manifest())
            self.assertNotEqual(runtime.identity(manifest()), runtime.identity(manifest() | {"version": "2"}))
            (Path(directory) / ".snitch-runtime.json").write_text("corrupt")
            with self.assertRaises(runtime.RuntimeFailed):
                runtime.model_binding(directory)

    def test_ready_survives_restart_without_download(self):
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            target = runtime.location(manifest(), env)
            (target / "bin").mkdir(parents=True)
            (target / "bin/python").touch()
            (target / ".ready").write_text(runtime.identity(manifest()))
            with patch.object(runtime.sys, "version_info", (3, 12)), patch.object(runtime.platform, "system", return_value="Linux"), patch.object(runtime.platform, "machine", return_value="x86_64"), patch.object(runtime, "_download") as download:
                self.assertEqual(runtime.install(manifest(), env, lambda _: self.fail("ready runtime changed")), target)
                download.assert_not_called()

    def test_missing_runtime_never_bootstraps_inference(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(runtime, "install") as install:
            with self.assertRaises(VoiceProviderUnavailable) as raised:
                voice_runtime.call(manifest(), directory, {"SNITCH_WORKER_MODELS_DIR": directory}, operation="verify")
            self.assertEqual(raised.exception.reason, "ml_libraries_missing")
            install.assert_not_called()

    def test_failed_install_keeps_previous_revision(self):
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            previous = runtime.location(manifest() | {"version": "previous"}, env)
            previous.mkdir(parents=True)
            (previous / ".ready").write_text("previous proof")
            with patch.object(runtime.sys, "version_info", (3, 12)), patch.object(runtime.platform, "system", return_value="Linux"), patch.object(runtime.platform, "machine", return_value="x86_64"), patch.object(runtime, "_download", side_effect=runtime.RuntimeFailed("hf_network")):
                with self.assertRaisesRegex(runtime.RuntimeFailed, "hf_network"):
                    runtime.install(manifest(), env, lambda _: None)
            self.assertEqual((previous / ".ready").read_text(), "previous proof")
            self.assertFalse(runtime.installed(manifest(), env))

    def test_environment_excludes_tokens_and_python_hooks(self):
        env = runtime.environment({"PATH": "/usr/bin", "HF_TOKEN": "secret", "SNITCH_HF_TOKEN_FILE": "/secret", "PYTHONPATH": "/evil", "LD_PRELOAD": "/evil", "PIP_INDEX_URL": "https://evil", "SNITCH_VOICE_DEVICE": "cpu"})
        self.assertEqual(env["HF_HUB_OFFLINE"], "1")
        for key in ["HF_TOKEN", "SNITCH_HF_TOKEN_FILE", "PYTHONPATH", "LD_PRELOAD", "PIP_INDEX_URL"]:
            self.assertNotIn(key, env)

    def test_unpacked_wheel_cannot_exceed_budget(self):
        with tempfile.TemporaryDirectory() as directory:
            with zipfile.ZipFile(Path(directory) / "a.whl", "w", compression=zipfile.ZIP_DEFLATED) as archive:
                archive.writestr("package/data", b"a" * 10000)
            with self.assertRaisesRegex(runtime.RuntimeFailed, "disk_full"):
                runtime._wheel_budget(Path(directory), 100)

    def test_download_checks_exact_bytes_and_hash(self):
        class Opener:
            def open(self, *args, **kwargs):
                return io.BytesIO(b"wheel")
        value = manifest()["wheels"][0] | {"size": 5, "sha256": hashlib.sha256(b"wheel").hexdigest()}
        with tempfile.TemporaryDirectory() as directory, patch.object(runtime.urllib.request, "build_opener", return_value=Opener()):
            runtime._download(value, Path(directory) / "valid")
            with self.assertRaises(runtime.RuntimeFailed):
                runtime._download(value | {"sha256": "0" * 64}, Path(directory) / "invalid")

    def test_legacy_readiness_delegates_only_to_requested_stage(self):
        from unittest.mock import Mock
        legacy = Mock()
        provider = voice_runtime.RuntimeProvider(legacy)
        with patch.object(provider, "_binding", return_value=("/legacy", None)):
            provider.ensure_diarization_ready({})
            legacy.ensure_diarization_ready.assert_called_once_with({})
            legacy.ensure_fingerprint_ready.assert_not_called()
            provider.ensure_fingerprint_ready({})
            legacy.ensure_fingerprint_ready.assert_called_once_with({})

    def test_validated_weights_with_missing_runtime_are_not_ready(self):
        from snitch_worker import voice_models
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            item = dict(model_id="test", repo="synthetic/model", revision="a" * 40, kind="fingerprint", engine="nemo-titanet")
            path = Path(voice_models.model_path(env, item["repo"], item["revision"]))
            path.mkdir(parents=True)
            (path / voice_models.READY_MARKER).write_text(item["revision"])
            runtime.bind(str(path), manifest())
            reports = []
            with patch.object(runtime, "install") as install:
                voice_models.ModelManager(env, lambda *a: self.fail("unexpected download"), lambda *a: self.fail("unexpected verification")).prepare(item, reports.append)
                install.assert_not_called()
            self.assertEqual(reports[-1]["state"], "failed")
            self.assertEqual(reports[-1]["cause_code"], "runtime_incompatible")

    def test_missing_runtime_revokes_known_ready_declaration_without_download(self):
        from unittest.mock import Mock
        from snitch_worker import voice_models
        with tempfile.TemporaryDirectory() as directory:
            env = {"SNITCH_WORKER_MODELS_DIR": directory}
            item = dict(model_id="test", repo="synthetic/model", revision="a" * 40, kind="fingerprint", engine="nemo-titanet")
            path = Path(voice_models.model_path(env, item["repo"], item["revision"]))
            path.mkdir(parents=True)
            runtime.bind(str(path), manifest())
            manager = voice_models.ModelManager(env, Mock(), Mock())
            http = Mock()
            with patch.object(runtime, "install") as install:
                voice_models._reconcile_held({"models": [item], "orders": [], "held": ["test"]}, manager, http)
                install.assert_not_called()
            self.assertEqual(http.post.call_args.kwargs["json"]["state"], "failed")
            manager._download.assert_not_called()
            manager._verify.assert_not_called()
