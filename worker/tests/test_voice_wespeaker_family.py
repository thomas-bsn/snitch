"""WeSpeaker family (#375, ADR 0056): one generic fingerprint adapter for speaker-embedding models exported as ONNX
(ResNet293-LM and its siblings), run by the on-demand engine. No WeSpeaker, torch or onnxruntime is needed here:
the adapter reaches the libraries through one small backend, replaced by a fake, and the engine is installed from
the offline wheelhouse. Synthetic values only."""

import json
import math
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_bridge, engine_runtime as runtime_module, voice_wespeaker
from snitch_worker.voice import VoiceProviderUnavailable

import wheelhouse
from test_engine_ondemand import OnDemandCase

DIMENSIONS = 256
CONFIG = "model: ResNet293\nmodel_args:\n  feat_dim: 80\n  embed_dim: 256\ndataset_args:\n  sample_rate: 16000\n  fbank_args:\n    num_mel_bins: 80\n    frame_length: 25\n    frame_shift: 10\n    dither: 1.0\n"


class FakeBackend:
    """Deterministic stand-in for torchaudio and onnxruntime: the 'model' returns a fixed vector of its dimensions."""

    def __init__(self, dimensions=DIMENSIONS, vector=None, seconds=6.0):
        self.dimensions, self.vector, self.seconds = dimensions, vector, seconds
        self.models, self.fbank_calls, self.decoded = [], [], []

    def decode(self, path):
        self.decoded.append(path)
        return object(), 16000, 1, int(self.seconds * 16000)

    def fbank(self, audio, start, end, options):
        self.fbank_calls.append((start, end, dict(options)))
        return ("features", start, end)

    def session(self, onnx_path):
        self.models.append(onnx_path)
        return lambda features: list(self.vector) if self.vector is not None else [1.0 + i for i in range(self.dimensions)]


class ModelDirectory(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name)

    def model(self, config=CONFIG, onnx="voxceleb_resnet293_LM.onnx", extra=()):
        (self.dir / onnx).write_bytes(b"\x08\x07synthetic-onnx")
        if config is not None:
            (self.dir / "config.yaml").write_text(config)
        for name in extra:
            (self.dir / name).write_bytes(b"x")
        return str(self.dir)

    def refusal(self, **kwargs):
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_wespeaker.speaker_loader(self.model(**kwargs), "cpu", backend=FakeBackend())
        self.assertEqual(caught.exception.reason, "model_load_failed")  # stable cause: the library text never travels
        return caught.exception.detail


class SafeguardTests(ModelDirectory):
    def test_only_an_onnx_graph_is_read_as_weights(self):
        for extra in ("avg_model.pt", "pytorch_model.bin", "model.ckpt", "weights.pkl"):
            with self.subTest(extra=extra):
                for old in self.dir.iterdir():
                    old.unlink()
                self.assertEqual(self.refusal(onnx="notes.txt", extra=(extra,)), "weights_format_unsafe")

    def test_weights_must_be_exactly_one_regular_file(self):
        self.assertEqual(self.refusal(onnx="a.onnx", extra=("b.onnx",)), "weights_ambiguous")
        empty = tempfile.TemporaryDirectory()
        self.addCleanup(empty.cleanup)
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_wespeaker.speaker_loader(empty.name, "cpu", backend=FakeBackend())
        self.assertEqual(caught.exception.detail, "weights_missing")

    def test_a_symlinked_graph_is_refused(self):
        target = self.dir / "elsewhere.bin"
        target.write_bytes(b"x")
        os.symlink(target, self.dir / "m.onnx")
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_wespeaker.speaker_loader(str(self.dir), "cpu", backend=FakeBackend())
        self.assertEqual(caught.exception.detail, "weights_unsafe")

    def test_an_oversized_graph_is_refused(self):
        with patch.object(voice_wespeaker, "MAX_WEIGHTS_BYTES", 4):
            self.assertEqual(self.refusal(), "weights_unsafe")

    def test_a_model_that_asks_for_remote_code_is_refused_before_anything_loads(self):
        backend = FakeBackend()
        for config in (CONFIG + "auto_map:\n  AutoModel: modeling.Custom\n", CONFIG + "trust_remote_code: true\n",
                       CONFIG + "boom: !!python/object/apply:os.system [id]\n"):
            with self.subTest(config=config[-30:]):
                (self.dir / "config.yaml").write_text(config)
                (self.dir / "m.onnx").write_bytes(b"x")
                with self.assertRaises(VoiceProviderUnavailable) as caught:
                    voice_wespeaker.speaker_loader(str(self.dir), "cpu", backend=backend)
                self.assertEqual(caught.exception.detail, "remote_code_refused")
        self.assertEqual(backend.models, [])

    def test_repository_python_files_are_never_imported(self):
        marker = self.dir / "ran"
        self.model(extra=())
        (self.dir / "modeling.py").write_text(f"open({str(marker)!r}, 'w').write('x')\n")
        (self.dir / "__init__.py").write_text(f"open({str(marker)!r}, 'w').write('x')\n")
        voice_wespeaker.speaker_loader(str(self.dir), "cpu", backend=FakeBackend())(str(self.dir / "a.wav"))
        self.assertFalse(marker.exists())

    def test_malformed_or_out_of_range_configuration_is_refused(self):
        cases = {"[unclosed": "config_invalid", "- a\n- b\n": "config_invalid",
                 "dataset_args:\n  fbank_args:\n    num_mel_bins: 100000\n": "config_invalid",
                 "dataset_args:\n  fbank_args:\n    num_mel_bins: [80]\n": "config_invalid",
                 "dataset_args:\n  sample_rate: 44100\n": "variant_unsupported"}
        for config, detail in cases.items():
            with self.subTest(config=config):
                self.assertEqual(self.refusal(config=config), detail)


class EmbeddingTests(ModelDirectory):
    def loader(self, backend=None, **kwargs):
        backend = backend or FakeBackend()
        return backend, voice_wespeaker.speaker_loader(self.model(**kwargs), "cpu", backend=backend)

    def test_a_window_is_embedded_with_the_fbank_of_the_configuration_and_no_dither(self):
        backend, open_audio = self.loader()
        audio = open_audio(str(self.dir / "a.wav"))
        self.assertEqual((audio.sample_rate, audio.channels, audio.num_samples), (16000, 1, 96000))
        vector = audio(0.5, 2.0)
        self.assertEqual(len(vector), DIMENSIONS)
        self.assertEqual(backend.fbank_calls, [(8000, 32000, {"num_mel_bins": 80, "frame_length": 25, "frame_shift": 10, "dither": 0.0})])
        self.assertEqual(backend.models, [str(self.dir / "voxceleb_resnet293_LM.onnx")])

    def test_without_a_configuration_the_wespeaker_defaults_apply(self):
        backend, open_audio = self.loader(config=None)
        open_audio("a.wav")(0.0, 1.0)
        self.assertEqual(backend.fbank_calls[0][2], {"num_mel_bins": 80, "frame_length": 25, "frame_shift": 10, "dither": 0.0})

    def test_a_non_finite_or_empty_or_oversized_vector_is_an_inference_failure(self):
        for vector in ([], [math.nan, 1.0], [math.inf] * 4, [0.0] * 8 + [1.0] * 5000):
            with self.subTest(size=len(vector)):
                _, open_audio = self.loader(backend=FakeBackend(vector=vector), onnx="m.onnx")
                with self.assertRaises(VoiceProviderUnavailable) as caught:
                    open_audio("a.wav")(0.0, 1.0)
                self.assertEqual(caught.exception.reason, "inference_failed")

    def test_the_fingerprint_follows_the_existing_contract(self):
        """Unit vectors, embedding_model, version and dimensions exactly as the other families (ADR 0044)."""
        env = {"SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "runtime-selftest"}
        _, open_audio = self.loader()
        provider = engine_bridge.SpeakerProvider("wespeaker", environ=env)
        diarization = {"job_id": "synthetic", "speakers": [{"label": "S0", "turns": [{"index": 0, "start": 0.0, "end": 2.0}, {"index": 1, "start": 2.0, "end": 3.0}]}]}
        config = {"provider": "pyannote", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
        with patch.object(provider, "_audio_opener", lambda config=None: open_audio):
            result = provider.fingerprint(config, diarization, "synthetic.wav", 0)
        self.assertEqual(result["dimensions"], DIMENSIONS)
        self.assertEqual(result["embedding_model_version"], "runtime-selftest")
        self.assertTrue(result["embedding_model"])
        self.assertEqual(len(result["fingerprints"]), 3)  # the speaker and its two turns
        for item in result["fingerprints"]:
            self.assertAlmostEqual(math.sqrt(sum(x * x for x in item["embedding"])), 1.0, places=6)


class BridgeTests(ModelDirectory):
    def request(self, operation, **extra):
        runtime = {"source": "on_demand", "family": "wespeaker", "engine": "wespeaker", "distribution": "wespeakerruntime", "version": "1.0.1",
                   "kind": "fingerprint", "kinds": ["fingerprint"], "python": "3.12", "platform": "linux-x86_64", "cuda": "",
                   "disk_bytes": 1 << 30, "vram_bytes": 0, "ram_bytes": 1 << 30, "estimated_seconds": 300}
        return {"manifest": runtime, "path": self.model(), "manifest_path": str(self.dir / "voice.json"), "operation": operation, **extra}

    def test_resnet293_verifies_and_fingerprints_through_the_wespeaker_engine(self):
        backend = FakeBackend()
        with patch.object(voice_wespeaker, "_backend", lambda: backend):
            self.assertEqual(engine_bridge.execute(self.request("verify")), {})
            diarization = {"job_id": "synthetic", "speakers": [{"label": "S0", "turns": [{"index": 0, "start": 0.0, "end": 2.0}]}]}
            config = {"provider": "pyannote", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
            result = engine_bridge.execute(self.request("fingerprint", config=config, diarization=diarization, audio=str(self.dir / "a.wav"), stream=0))
        self.assertEqual(result["dimensions"], DIMENSIONS)
        self.assertTrue(result["fingerprints"])
        self.assertEqual(backend.models[0], str(self.dir / "voxceleb_resnet293_LM.onnx"))

    def test_a_refused_model_ends_with_a_stable_cause_in_the_result_file(self):
        incoming, outgoing = self.dir / "in.json", self.dir / "out.json"
        request = self.request("verify")
        (self.dir / "voxceleb_resnet293_LM.onnx").unlink()
        (self.dir / "avg_model.pt").write_bytes(b"x")
        incoming.write_text(json.dumps(request))
        with patch.object(engine_bridge.sys, "argv", ["engine_bridge", str(incoming), str(outgoing)]), patch.object(voice_wespeaker, "_backend", FakeBackend):
            engine_bridge.main()
        self.assertEqual(json.loads(outgoing.read_text()), {"error": "model_load_failed", "detail": "weights_format_unsafe"})

    def test_the_pyannote_embedded_resnet34_is_untouched(self):
        manifest = {"engine": "pyannote", "kind": "fingerprint"}
        with patch("snitch_worker.engine_bridge.PyannoteProvider") as pyannote, patch("snitch_worker.engine_bridge.SpeakerProvider") as speaker, \
                patch("snitch_worker.voice_selftest.embed_synthetic"):
            engine_bridge.execute({"manifest": manifest, "path": "/m", "manifest_path": "/x/voice.json", "operation": "verify"})
        pyannote.assert_called_once()
        speaker.assert_not_called()


class FamilyRuntimeTests(OnDemandCase):
    def runtime(self, **changes):
        value = {"source": "on_demand", "family": "wespeaker", "engine": "wespeaker", "distribution": "wespeakerruntime", "version": "1.0.1",
                 "kind": "fingerprint", "kinds": ["fingerprint"], "python": "3.12", "platform": "linux-x86_64", "cuda": "",
                 "disk_bytes": 1 << 20, "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 300}
        value.update(changes)
        return value

    def setUp(self):
        super().setUp()
        wheelhouse.build_wespeaker_wheelhouse(Path(self.directory.name) / "wheels")
        self.env["SNITCH_ENGINE_FIND_LINKS"] = str(Path(self.directory.name) / "wheels")

    def test_the_family_installs_from_the_wheelhouse_with_its_probe_and_a_freeze(self):
        target = self.install(self.runtime())
        self.assertIn("wespeakerruntime==1.0.1", (target / "pip-freeze.txt").read_text().splitlines())
        self.assertTrue(runtime_module.installed(self.runtime(), self.env))
        self.assertEqual([i["engine"] for i in runtime_module.inventory(self.env)], ["wespeaker"])

    def test_the_family_is_closed_and_fingerprint_only(self):
        runtime_module.validate_runtime(self.runtime())
        for changes in ({"kind": "diarization"}, {"kinds": ["fingerprint", "diarization"]}, {"distribution": "wespeaker"},
                        {"distribution": "git+https://example.invalid/wespeaker.git"}, {"engine": "nemo-toolkit"},
                        {"version": "--index-url=http://evil"}, {"version": "1.0.1 --pre"}, {"cuda": "12.8"}):
            with self.subTest(changes=changes), self.assertRaises(runtime_module.RuntimeFailed):
                runtime_module.validate_runtime(self.runtime(**changes))

    def test_a_missing_version_is_package_not_found(self):
        self.assertEqual(self.failure(self.runtime(version="9.9.9")), "package_not_found")

    def test_pip_asks_for_the_named_distribution_only(self):
        seen = []
        self.env["HF_TOKEN"] = "secret"
        real = runtime_module._pip
        with patch.object(runtime_module, "_pip", lambda python, arguments, env, timeout: (seen.append((arguments, dict(env))), real(python, arguments, env, timeout))[1]):
            self.install(self.runtime())
        install = next(arguments for arguments, _ in seen if arguments[0] == "install")
        self.assertEqual(install[-1], "wespeakerruntime==1.0.1")
        self.assertFalse([a for a in install if a.startswith(("git+", "http", "-e", "--trusted", "--pre"))])
        for _, env in seen:
            self.assertNotIn("HF_TOKEN", env)


if __name__ == "__main__":
    unittest.main()
