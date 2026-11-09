"""Generic NeMo family adapter (#373, ADR 0056): diarization and fingerprint run by the Python of an on-demand
environment through engine_bridge, with the fixed safeguards. No NeMo, torch or network: fakes stand in for the
library and the weights; the real-GPU proof is a separate ticket (#377). Synthetic values only."""

import io
import json
import os
import sys
import tarfile
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_bridge, voice, voice_manifest, voice_models, voice_nemo, voice_nemo_family, voice_runtime, voice_selftest

REPO, REVISION = "nvidia/synthetic-nemo-model", "d" * 40
SPEAKERS = "synthetic/titanet-like"


def runtime(kind="diarization"):
    return {"source": "on_demand", "family": "nemo", "engine": "nemo-toolkit", "distribution": "nemo-toolkit", "version": "3.0.0",
            "kind": kind, "kinds": ["diarization", "fingerprint"], "python": "3.12", "platform": "linux-x86_64", "cuda": "",
            "disk_bytes": 1 << 20, "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 600}


CONFIG_YAML = """\
target: nemo.collections.asr.models.SortformerEncLabelModel
sample_rate: 16000
encoder:
  _target_: nemo.collections.asr.modules.ConformerEncoder
  feat_in: 80
  rate: ${sample_rate}
"""


def write_archive(directory, config=CONFIG_YAML, weights=b"weights", extra=None):
    path = Path(directory) / "model.nemo"
    with tarfile.open(path, "w") as archive:
        for name, data in {"model_config.yaml": config.encode(), "model_weights.ckpt": weights, **(extra or {})}.items():
            info = tarfile.TarInfo(name)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
    return str(path)


def accepting(raw):
    return None


def rejecting(raw):
    raise voice_nemo_family.FamilyRefused("weights_unsafe")


class RefusalTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = self.tmp.name
        self.addCleanup(self.tmp.cleanup)

    def refused(self, detail, weights=accepting):
        with self.assertRaises(voice_nemo_family.FamilyRefused) as caught:
            voice_nemo_family.preflight(self.dir, load_weights=weights)
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, detail)

    def test_a_clean_model_directory_is_accepted(self):
        archive = write_archive(self.dir)
        (Path(self.dir) / "README.md").write_text("notes")
        self.assertEqual(voice_nemo_family.preflight(self.dir, load_weights=accepting), archive)

    def test_python_files_of_the_repository_mean_remote_code(self):
        write_archive(self.dir)
        (Path(self.dir) / "modeling_nemotron.py").write_text("raise SystemExit")
        self.refused("remote_code_required")

    def test_a_config_asking_for_remote_code_is_refused(self):
        write_archive(self.dir)
        (Path(self.dir) / "config.json").write_text(json.dumps({"auto_map": {"AutoModel": "modeling_x.X"}}))
        self.refused("remote_code_required")
        (Path(self.dir) / "config.json").write_text(json.dumps({"trust_remote_code": True}))
        self.refused("remote_code_required")

    def test_a_repository_file_is_never_imported(self):
        write_archive(self.dir)
        (Path(self.dir) / "benign_config.json").write_text("{}")
        sentinel = Path(self.dir) / "ran"
        (Path(self.dir) / "hook.pth").write_text(f"import pathlib; pathlib.Path({str(sentinel)!r}).write_text('x')")
        voice_nemo_family.preflight(self.dir, load_weights=accepting)
        self.assertFalse(sentinel.exists())

    def test_configuration_is_data_only(self):
        for config in (CONFIG_YAML.replace("nemo.collections.asr.modules.ConformerEncoder", "os.system"),
                       CONFIG_YAML + "extra: ${oc.env:HF_TOKEN}\n",
                       CONFIG_YAML + "nested:\n  target: os.system\n",
                       CONFIG_YAML.replace("target: nemo.collections.asr.models.SortformerEncLabelModel", "target: evil.Model"),
                       CONFIG_YAML + "boom: !!python/object/apply:os.system [id]\n"):
            with self.subTest(config=config), tempfile.TemporaryDirectory() as directory:
                self.dir = directory
                write_archive(directory, config)
                self.refused("config_executable" if "python/object" not in config else "config_unreadable")

    def test_unsafe_weights_are_refused(self):
        write_archive(self.dir)
        self.refused("weights_unsafe", weights=rejecting)

    def test_the_real_check_loads_weights_in_safe_mode_only(self):
        calls = []
        fake = types.ModuleType("torch")
        fake.load = lambda stream, **kw: calls.append(kw) or {}
        with patch.dict(sys.modules, {"torch": fake}):
            voice_nemo_family.check_weights(b"x")
        self.assertEqual(calls, [{"map_location": "cpu", "weights_only": True}])
        broken = types.ModuleType("torch")

        def refuse(stream, **kw):
            raise RuntimeError("Unsupported global: os.system")
        broken.load = refuse
        with patch.dict(sys.modules, {"torch": broken}):
            with self.assertRaises(voice_nemo_family.FamilyRefused) as caught:
                voice_nemo_family.check_weights(b"x")
        self.assertEqual(caught.exception.detail, "weights_unsafe")
        self.assertNotIn("os.system", str(caught.exception))

    def test_archive_problems_keep_their_existing_stable_codes(self):
        self.refused("archive_missing")
        write_archive(self.dir, extra={"../escape": b"x"})
        self.refused("archive_unsafe")


class LoaderTests(unittest.TestCase):
    def test_a_refused_model_never_reaches_the_library(self):
        with tempfile.TemporaryDirectory() as directory:
            write_archive(directory)
            (Path(directory) / "modeling_x.py").write_text("")
            restored = []
            with self.assertRaises(voice_nemo_family.FamilyRefused):
                voice_nemo_family.load_diarizer(directory, "cpu", restore=lambda *a: restored.append(a), load_weights=accepting)
        self.assertEqual(restored, [])

    def test_an_accepted_model_is_restored_from_its_archive_on_the_device(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = write_archive(directory)
            seen = []

            def restore(path, device):
                seen.append((path, device))
                return types.SimpleNamespace(diarize=lambda audio, batch_size: [["0.0 1.0 speaker_0"]])
            run = voice_nemo_family.load_diarizer(directory, "cpu", restore=restore, load_weights=accepting)
            self.assertEqual(run("/a.wav"), ["0.0 1.0 speaker_0"])
        self.assertEqual(seen, [(archive, "cpu")])


def model_dir(root, kind, files=None):
    """A ready model on the volume as the Worker lays it out, and the environment that finds it."""
    env = {"SNITCH_WORKER_MODELS_DIR": root, "SNITCH_VOICE_DEVICE": "cpu"}
    manifest_path = voice_manifest.manifest_path(env)
    path = voice_models.model_path(env, REPO, REVISION)
    os.makedirs(path)
    for name, data in (files or {}).items():
        Path(path, name).write_text(data)
    Path(path, voice_models.READY_MARKER).write_text("")
    return env, path, manifest_path


class BridgeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def test_diarization_runs_through_the_bridge_with_the_existing_turn_contract(self):
        env, path, manifest_path = model_dir(self.tmp.name, "diarization")
        write_archive(path)
        config = {"provider": "nemo", "model": REPO, "model_version": REVISION, "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
        audio = os.path.join(self.tmp.name, "a.wav")
        voice_selftest.write_synthetic_audio(audio)
        fake = types.SimpleNamespace(diarize=lambda audio, batch_size: [["2.0 3.0 speaker_1", "0.0 1.5 speaker_0"]])
        with patch.dict(os.environ, env), patch.object(voice_nemo_family, "check_weights", accepting), \
                patch.object(voice_nemo_family, "_restore", lambda archive, device: fake):
            result = engine_bridge.execute({"manifest": runtime("diarization"), "path": path, "manifest_path": manifest_path,
                                            "operation": "diarize", "config": config, "audio": audio, "stream": 1})
            self.assertEqual(engine_bridge.execute({"manifest": runtime("diarization"), "path": path, "manifest_path": manifest_path,
                                                    "operation": "verify"}), {})
        self.assertEqual(result["audio_stream_index"], 1)
        self.assertEqual([s["label"] for s in result["speakers"]], ["SPEAKER_00", "SPEAKER_01"])
        self.assertEqual(result["speakers"][0]["turns"], [{"start": 0.0, "end": 1.5}])

    def test_a_remote_code_model_is_refused_by_the_bridge_with_a_stable_code(self):
        env, path, manifest_path = model_dir(self.tmp.name, "diarization")
        write_archive(path)
        Path(path, "modeling_nemotron.py").write_text("raise SystemExit")
        incoming, outgoing = os.path.join(self.tmp.name, "in.json"), os.path.join(self.tmp.name, "out.json")
        Path(incoming).write_text(json.dumps({"manifest": runtime("diarization"), "path": path, "manifest_path": manifest_path, "operation": "verify"}))
        saved, sys.argv = sys.argv, ["engine_bridge", incoming, outgoing]
        try:
            with patch.dict(os.environ, env):
                engine_bridge.main()
        finally:
            sys.argv = saved
        answer = json.loads(Path(outgoing).read_text())
        self.assertEqual(answer, {"error": "model_load_failed", "detail": "remote_code_required"})
        # The Worker maps that reason, as for every refused model, onto its existing model cause.
        self.assertEqual(voice_models._cause_of_verification(voice.VoiceProviderUnavailable(answer["error"])), "runtime_incompatible")

    def test_fingerprint_follows_the_existing_contract(self):
        env, path, manifest_path = model_dir(self.tmp.name, "fingerprint")
        environ = dict(env, SNITCH_VOICE_EMBEDDING_MODEL_VERSION="x")
        audio = os.path.join(self.tmp.name, "a.wav")
        voice_selftest.write_synthetic_audio(audio, seconds=6.0)
        from snitch_worker.voice_pyannote import DecodedAudio

        def loader(model_path, device):
            self.assertEqual(model_path, path)
            return lambda audio_path: DecodedAudio(lambda start, end: [3.0, 4.0, 0.0, 0.0], 16000, 1, 16000 * 6)
        provider = voice_nemo_family.provider(runtime("fingerprint"), environ, load_embedder=loader, device_probe=lambda: "cpu")
        diarization = {"job_id": 3, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 4.0}]}]}
        config = {"provider": "pyannote", "min_fingerprint_seconds": 1.0, "min_quality": 0.4,
                  "fingerprint_model": {"engine": "nemo-titanet", "repo": REPO, "revision": REVISION}}
        result = provider.fingerprint(config, diarization, audio, 0)
        self.assertEqual((result["embedding_model"], result["embedding_model_version"], result["dimensions"]), (REPO, REVISION, 4))
        for item in result["fingerprints"]:
            self.assertAlmostEqual(sum(x * x for x in item["embedding"]) ** 0.5, 1.0, places=6)
            self.assertEqual([round(x, 6) for x in item["embedding"]], [0.6, 0.8, 0.0, 0.0])

    def test_fingerprint_models_use_the_fixed_titanet_loader(self):
        provider = voice_nemo_family.provider(runtime("fingerprint"), {})
        self.assertEqual(provider.engine, "nemo-titanet")
        with patch.dict(sys.modules, {"torch": None, "torchaudio": None}), self.assertRaises(ImportError):
            provider._load_embedder("/m", "cpu")  # the loader of voice_titanet: archive and weights are data, never code

    def test_an_unknown_kind_is_refused(self):
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            voice_nemo_family.provider({**runtime(), "kind": "separation"}, {})
        self.assertEqual(caught.exception.reason, "model_load_failed")


class RuntimeCallTests(unittest.TestCase):
    """voice_runtime.call accepts an on-demand description and runs the venv's Python with the fixed bridge."""

    def test_the_call_uses_the_environment_python_and_hands_the_runtime_description(self):
        commands = []

        def run(command, **kwargs):
            commands.append((command, kwargs))
            Path(command[-1]).write_text(json.dumps({"result": {}}))
        with tempfile.TemporaryDirectory() as root, patch.object(voice_runtime.engine_runtime, "installed", return_value=True), \
                patch.object(voice_runtime.engine_runtime, "location", return_value=Path(root) / "venv"), \
                patch.object(voice_runtime.cancellation, "run_process", run), patch.object(voice_runtime.engine_runtime, "gpu_state", return_value=None):
            voice_runtime.call(runtime("diarization"), root, {"SNITCH_WORKER_MODELS_DIR": root}, operation="verify")
        command, kwargs = commands[0]
        self.assertEqual(command[0], str(Path(root) / "venv/bin/python"))
        self.assertIn("-I", command)
        self.assertNotIn("HF_TOKEN", kwargs["env"])

    def test_a_gpu_with_too_little_memory_stops_before_any_inference(self):
        needy = {**runtime(), "vram_bytes": 8 << 30}
        with patch.object(voice_runtime.engine_runtime, "gpu_state", return_value=("12.8", 24 << 30, 1 << 30)):
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                voice_runtime.inference_device(needy, {})
        self.assertEqual(caught.exception.reason, "gpu_out_of_memory")


class EmbeddedNemotronPathTests(unittest.TestCase):
    """What happened to nvidia/Nemotron-3-Diarization before on-demand engines (synthetic reproduction of the path).

    A catalogue entry classified `nemo` has no runtime: the Worker verifies it in-process with the NeMo of its own image.
    Whatever goes wrong there, the user only ever saw `runtime_incompatible`."""

    def provider(self, root, loader):
        env = {"SNITCH_VOICE_EMBEDDING_MODEL_PATH": root, "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "v", "SNITCH_VOICE_DIARIZATION_MODEL_PATH": root}
        return voice_nemo.NemoProvider(environ=env, load_pipeline=loader, device_probe=lambda: "cpu")

    def verify(self, provider):
        try:
            voice_selftest.diarize_synthetic(provider, "nemo")
        except voice.VoiceProviderUnavailable as error:
            return voice_models._cause_of_verification(error), error.reason
        return None

    def test_a_model_of_another_class_than_sortformer_fails_as_runtime_incompatible(self):
        def loader(path, device):
            raise RuntimeError("Can't instantiate NemotronDiarizer: unknown target")  # restore_from of a newer model class
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(self.verify(self.provider(root, loader)), ("runtime_incompatible", "model_load_failed"))

    def test_a_worker_image_without_nemo_fails_as_runtime_incompatible(self):
        def loader(path, device):
            raise ImportError("No module named nemo")
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(self.verify(self.provider(root, loader)), ("runtime_incompatible", "ml_libraries_missing"))


if __name__ == "__main__":
    unittest.main()
