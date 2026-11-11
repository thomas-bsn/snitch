"""SepFormer separation adapter of the on-demand SpeechBrain engine (#401, ADR 0065).

The safeguards of the family apply: no repository code, the hyperparams YAML is data (never HyperPyYAML), every
checkpoint is scanned and read with weights_only, nothing is fetched. torch and SpeechBrain are replaced by fakes;
the tests that run the real library on random weights are skipped when it is not installed, and never download a
model. The two YAML fixtures are the published inference files of speechbrain/sepformer-whamr16k and
speechbrain/sepformer-libri3mix (Apache-2.0). Synthetic values only."""
import collections
import importlib.util
import json
import os
import pickle
import sys
import tempfile
import types
import unittest
import wave
import zipfile
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_bridge, voice_selftest, voice_sepformer
from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_ecapa import UnsupportedVariant

FIXTURES = Path(__file__).parent / "fixtures" / "sepformer"
TWO_VOICES = (FIXTURES / "hyperparams.yaml").read_text()
THREE_VOICES = (FIXTURES / "hyperparams-3-voices.yaml").read_text()
HAVE_YAML = importlib.util.find_spec("yaml") is not None
HAVE_SPEECHBRAIN = HAVE_YAML and all(importlib.util.find_spec(name) is not None for name in ("torch", "speechbrain"))
CHECKPOINTS = ("encoder.ckpt", "masknet.ckpt", "decoder.ckpt")


def write_checkpoint(path, payload=None):
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("archive/data.pkl", pickle.dumps(collections.OrderedDict(a=1) if payload is None else payload, protocol=2))
        archive.writestr("archive/version", "3\n")


class FakeLibrary:
    """Stands for torch and the SpeechBrain dual-path classes; records what was built and how weights were read."""

    def __init__(self):
        self.loads = []
        self.built = []
        self.strict = []
        library = self

        def model(name):
            class Model:
                def __init__(self, **kwargs):
                    library.built.append((name, kwargs))

                def to(self, *_):
                    return self

                def eval(self):
                    return self

                def load_state_dict(self, state, strict):
                    library.strict.append(strict)

            return Model

        torch = types.ModuleType("torch")
        torch.load = lambda path, **kwargs: library.loads.append((Path(path).name, kwargs)) or {}
        dual_path = types.ModuleType("speechbrain.lobes.models.dual_path")
        for name in ("Encoder", "SBTransformerBlock", "Dual_Path_Model", "Decoder"):
            setattr(dual_path, name, model(name))
        self.modules = {"torch": torch, "speechbrain.lobes.models.dual_path": dual_path}
        for name in ("speechbrain", "speechbrain.lobes", "speechbrain.lobes.models"):
            self.modules[name] = types.ModuleType(name)
        self.modules["speechbrain.lobes.models"].dual_path = dual_path


@unittest.skipUnless(HAVE_YAML, "PyYAML is required")
class LoadTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name) / "model"
        self.dir.mkdir()
        self.marker = Path(self.tmp.name) / "pwned"
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES)
        for name in CHECKPOINTS:
            write_checkpoint(self.dir / name)

    def load(self):
        library = FakeLibrary()
        with patch.dict(sys.modules, library.modules):
            return voice_sepformer.load(str(self.dir), "cpu"), library

    def refused(self, detail):
        library = FakeLibrary()
        with patch.dict(sys.modules, library.modules), self.assertRaises(UnsupportedVariant) as caught:
            voice_sepformer.load(str(self.dir), "cpu")
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, detail)
        self.assertFalse(self.marker.exists())
        return library

    def test_a_two_voice_16_khz_model_is_built_from_data_and_read_with_weights_only(self):
        separator, library = self.load()
        self.assertEqual((separator.sample_rate, separator.num_sources), (16000, 2))
        self.assertTrue(callable(separator.separate))
        self.assertEqual(sorted(name for name, _ in library.loads), sorted(CHECKPOINTS))
        self.assertTrue(all(kwargs.get("weights_only") is True for _, kwargs in library.loads))
        self.assertEqual(library.strict, [True, True, True])
        built = dict((name, kwargs) for name, kwargs in library.built if name != "SBTransformerBlock")
        self.assertEqual(built["Encoder"], {"kernel_size": 16, "out_channels": 256})
        self.assertEqual(built["Decoder"], {"in_channels": 256, "out_channels": 1, "kernel_size": 16, "stride": 8, "bias": False})
        masknet = dict(built["Dual_Path_Model"])
        self.assertEqual({type(masknet.pop("intra_model")).__name__, type(masknet.pop("inter_model")).__name__}, {"Model"})
        self.assertEqual(masknet, {"num_spks": 2, "in_channels": 256, "out_channels": 256, "num_layers": 2, "K": 250, "norm": "ln",
                                   "linear_layer_after_inter_intra": False, "skip_around_intra": True})
        blocks = [kwargs for name, kwargs in library.built if name == "SBTransformerBlock"]
        self.assertEqual(blocks, [{"num_layers": 8, "d_model": 256, "nhead": 8, "d_ffn": 1024, "dropout": 0,
                                   "use_positional_encoding": True, "norm_before": True}] * 2)

    def test_the_limits_of_a_three_voice_8_khz_model_are_read_from_its_own_file(self):
        (self.dir / "hyperparams.yaml").write_text(THREE_VOICES)
        separator, library = self.load()
        self.assertEqual((separator.sample_rate, separator.num_sources), (8000, 3))
        self.assertEqual(dict(library.built)["Dual_Path_Model"]["num_spks"], 3)

    def test_training_leftovers_of_the_repository_are_tolerated_and_never_read(self):
        for name in ("optimizer.ckpt", "brain.ckpt", "counter.ckpt", "lr_scheduler.ckpt", "dataloader-TRAIN.ckpt", "CKPT.yaml",
                     "hyperparams_train.yaml", "config.json", "README.md", "test_mixture16k.wav", ".snitch-ready", ".snitch-runtime.json"):
            (self.dir / name).write_bytes(b"not a checkpoint, never opened")
        _, library = self.load()
        self.assertEqual(sorted(name for name, _ in library.loads), sorted(CHECKPOINTS))

    def test_repository_code_links_directories_and_unknown_files_are_refused(self):
        (self.dir / "custom.py").write_text("print('remote code')")
        self.refused("code_file_in_model")
        (self.dir / "custom.py").unlink()
        (self.dir / "weights.safetensors").write_bytes(b"x")
        self.refused("unexpected_file")
        (self.dir / "weights.safetensors").unlink()
        (self.dir / "link.ckpt").symlink_to(self.dir / "encoder.ckpt")
        self.refused("unexpected_directory_or_link")
        (self.dir / "link.ckpt").unlink()
        (self.dir / "metadata").mkdir()
        self.refused("unexpected_directory_or_link")

    def test_a_missing_checkpoint_is_refused(self):
        (self.dir / "masknet.ckpt").unlink()
        self.refused("model_file_missing")

    def test_a_hyperparams_program_is_read_as_data_and_refused(self):
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES + "\nboom: !apply:os.system [touch " + str(self.marker) + "]\n")
        self.refused("hyperparams_tag_refused")
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES + "\nboom: !!python/object/apply:os.system [touch " + str(self.marker) + "]\n")
        self.refused("hyperparams_tag_refused")

    def test_another_architecture_or_an_unknown_parameter_is_refused(self):
        for text in (TWO_VOICES.replace("dual_path.Encoder", "conv_tasnet.Encoder"),
                     TWO_VOICES.replace("  K: 250\n", "  K: 250\n  surprise: 1\n"),
                     TWO_VOICES.replace("  K: 250\n", ""),
                     TWO_VOICES.replace("  K: 250\n", "  K: many\n"),
                     TWO_VOICES.replace("intra_model: !ref <SBtfintra>", "intra_model: !ref <SBtfinter>"),
                     TWO_VOICES.replace("  out_channels: 1\n", "  out_channels: 2\n")):
            (self.dir / "hyperparams.yaml").write_text(text)
            with self.subTest(text=text[-200:]):
                self.assertEqual(self.refused("unsupported_architecture").built, [])

    def test_limits_outside_what_the_adapter_supports_are_refused(self):
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES.replace("sample_rate: 16000", "sample_rate: 44100"))
        self.refused("sample_rate_unsupported")
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES.replace("num_spks: 2", "num_spks: 1"))
        self.refused("source_count_unsupported")
        (self.dir / "hyperparams.yaml").write_text(TWO_VOICES.replace("num_spks: 2", "num_spks: 9"))
        self.refused("source_count_unsupported")
        # The mask network must separate the number of voices the file declares.
        (self.dir / "hyperparams.yaml").write_text(THREE_VOICES.replace("num_spks: 3\n\n", "num_spks: 2\n\n", 1))
        self.refused("unsupported_architecture")

    def test_a_malicious_checkpoint_is_refused_and_never_executed(self):
        class Trap:
            def __reduce__(self):
                return (os.system, ("touch " + str(self_marker),))
        self_marker = self.marker
        write_checkpoint(self.dir / "decoder.ckpt", Trap())
        library = self.refused("checkpoint_forbidden_object")
        self.assertEqual(library.loads, [])

    def test_missing_libraries_are_an_explicit_state(self):
        with patch.dict(sys.modules, {"torch": None}), self.assertRaises(VoiceProviderUnavailable) as caught:
            voice_sepformer.load(str(self.dir), "cpu")
        self.assertEqual(caught.exception.reason, "ml_libraries_missing")


class DeviceTests(unittest.TestCase):
    def test_the_device_follows_the_worker_setting_and_never_degrades_silently(self):
        self.assertEqual(voice_sepformer.resolve_device({"SNITCH_VOICE_DEVICE": "cpu"}, probe=lambda: "cuda"), "cpu")
        self.assertEqual(voice_sepformer.resolve_device({}, probe=lambda: "cuda"), "cuda")
        self.assertEqual(voice_sepformer.resolve_device({"SNITCH_VOICE_DEVICE": "auto"}, probe=lambda: "cpu"), "cpu")
        for environ, reason in (({"SNITCH_VOICE_DEVICE": "cuda"}, "cuda_unavailable"), ({"SNITCH_VOICE_DEVICE": "tpu"}, "device_invalid")):
            with self.assertRaises(VoiceProviderUnavailable) as caught:
                voice_sepformer.resolve_device(environ, probe=lambda: "cpu")
            self.assertEqual(caught.exception.reason, reason)


class BridgeTests(unittest.TestCase):
    """The fixed protocol a SpeechBrain environment runs: a test inference, or the separation of one window."""

    def setUp(self):
        patcher = patch.dict(os.environ, {"SNITCH_VOICE_DEVICE": "cpu"})
        patcher.start()
        self.addCleanup(patcher.stop)
        self.separator = voice_sepformer.Separator(16000, 2, None)

    def request(self, operation, **extra):
        runtime = {"source": "on_demand", "family": "speechbrain", "engine": "speechbrain", "distribution": "speechbrain",
                   "version": "1.0.3", "kind": "separation", "kinds": ["fingerprint", "separation"], "python": "3.12", "platform": "linux-x86_64",
                   "cuda": "", "disk_bytes": 1, "vram_bytes": 0, "ram_bytes": 0, "estimated_seconds": 1}
        return {"manifest": runtime, "path": "/model", "manifest_path": "/models/voice.json", "operation": operation, **extra}

    def test_the_test_inference_loads_the_model_and_separates_a_synthetic_mixture(self):
        proofs = []
        with patch.object(voice_sepformer, "load", return_value=self.separator) as load, \
                patch.object(voice_sepformer, "separate_file", return_value={"sample_rate": 16000, "sources": []}) as separate, \
                patch.object(voice_selftest, "separate_synthetic", side_effect=lambda run, sources, rate: proofs.append((run("/mix.wav", "/out"), sources, rate))):
            self.assertEqual(engine_bridge.execute(self.request("verify")), {})
        load.assert_called_once_with("/model", "cpu")
        separate.assert_called_once_with(self.separator, "/mix.wav", "/out")
        self.assertEqual(proofs, [({"sample_rate": 16000, "sources": []}, 2, 16000)])

    def test_a_failed_test_inference_is_reported_with_its_stable_reason(self):
        with tempfile.TemporaryDirectory() as directory:
            incoming, outgoing = Path(directory) / "in.json", Path(directory) / "out.json"
            incoming.write_text(json.dumps(self.request("verify")))
            # A model that answers a single source for two voices does not pass the proof.
            one = lambda separator, audio, output: {"sample_rate": 16000, "sources": [{"index": 0, "path": audio, "samples": 32000}]}  # noqa: E731
            with patch.object(voice_sepformer, "load", return_value=self.separator), patch.object(voice_sepformer, "separate_file", side_effect=one), \
                    patch.object(sys, "argv", ["engine_bridge", str(incoming), str(outgoing)]):
                engine_bridge.main()
            self.assertEqual(json.loads(outgoing.read_text()), {"error": "inference_failed"})

    def test_a_separation_call_returns_the_sources_of_the_window_asked(self):
        answer = {"sample_rate": 16000, "start": 1.0, "end": 4.0, "gain": 1.0, "sources": [{"index": 0, "path": "/out/source-0.wav", "samples": 48000}]}
        with patch.object(voice_sepformer, "load", return_value=self.separator), patch.object(voice_sepformer, "separate_file", return_value=answer) as separate:
            result = engine_bridge.execute(self.request("infer", audio="/clip.wav", output_dir="/out", start=1.0, end=4.0))
        self.assertEqual(result, answer)
        separate.assert_called_once_with(self.separator, "/clip.wav", "/out", 1.0, 4.0)

    def test_only_the_speechbrain_family_runs_separation(self):
        request = self.request("verify")
        request["manifest"] = {**request["manifest"], "family": "nemo", "engine": "nemo-toolkit"}
        with patch.object(voice_sepformer, "load") as load, self.assertRaises(VoiceProviderUnavailable) as caught:
            engine_bridge.execute(request)
        self.assertEqual((caught.exception.reason, caught.exception.detail), ("model_load_failed", "kind_not_supported"))
        load.assert_not_called()


@unittest.skipUnless(HAVE_SPEECHBRAIN, "torch and SpeechBrain are required")
class RealLibraryTests(unittest.TestCase):
    """The real SpeechBrain classes on random weights saved by this test: nothing is downloaded. It proves that the
    architecture read from the published YAML builds, that its own checkpoints pass the scan and load strictly, and
    that a synthetic mixture comes out as the declared number of voices. It says nothing of separation quality."""

    @classmethod
    def setUpClass(cls):
        import torch
        import yaml
        from speechbrain.lobes.models import dual_path

        cls.tmp = tempfile.TemporaryDirectory()
        cls.dir = Path(cls.tmp.name) / "model"
        cls.dir.mkdir()
        # A small copy of the published architecture keeps the test fast; the published one is built in one test below.
        small = TWO_VOICES.replace("num_layers: 8", "num_layers: 1").replace("d_ffn: 1024", "d_ffn: 64").replace("K: 250", "K: 50")
        (cls.dir / "hyperparams.yaml").write_text(small)
        architecture = voice_sepformer.read_architecture(voice_sepformer.scan_hyperparams(small))
        torch.manual_seed(7)
        modules = {"encoder": dual_path.Encoder(**architecture["encoder"]), "decoder": dual_path.Decoder(**architecture["decoder"]),
                   "masknet": dual_path.Dual_Path_Model(intra_model=dual_path.SBTransformerBlock(**architecture["intra"]),
                                                        inter_model=dual_path.SBTransformerBlock(**architecture["inter"]), **architecture["masknet"])}
        for name, module in modules.items():
            torch.save(module.state_dict(), cls.dir / (name + ".ckpt"))
        del yaml

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def test_the_published_architecture_builds_with_the_real_classes(self):
        from speechbrain.lobes.models import dual_path
        architecture = voice_sepformer.read_architecture(voice_sepformer.scan_hyperparams(TWO_VOICES))
        dual_path.Dual_Path_Model(intra_model=dual_path.SBTransformerBlock(**architecture["intra"]),
                                  inter_model=dual_path.SBTransformerBlock(**architecture["inter"]), **architecture["masknet"])

    def test_a_tensor_mixture_gives_one_source_per_voice_of_the_same_length(self):
        import torch
        separator = voice_sepformer.load(str(self.dir), "cpu")
        mixture = torch.sin(torch.arange(16000) * 0.05).unsqueeze(0)
        sources = separator.separate(mixture)
        self.assertEqual(tuple(sources.shape), (2, 16000))
        self.assertTrue(bool(torch.isfinite(sources).all()))

    def test_the_test_inference_separates_a_synthetic_mixture_file(self):
        separator = voice_sepformer.load(str(self.dir), "cpu")
        voice_selftest.separate_synthetic(lambda audio, out: voice_sepformer.separate_file(separator, audio, out),
                                          separator.num_sources, separator.sample_rate)

    def test_a_window_of_a_44_1_khz_stereo_file_is_written_at_the_rate_of_the_model(self):
        separator = voice_sepformer.load(str(self.dir), "cpu")
        with tempfile.TemporaryDirectory() as directory:
            audio = os.path.join(directory, "mix.wav")
            voice_selftest.write_synthetic_mixture(audio, seconds=3.0, rate=44100, channels=2)
            result = voice_sepformer.separate_file(separator, audio, directory, start=0.5, end=2.0)
            self.assertEqual((result["sample_rate"], result["start"], result["end"], len(result["sources"])), (16000, 0.5, 2.0, 2))
            for index, source in enumerate(result["sources"]):
                self.assertEqual((source["index"], source["samples"]), (index, 24000))
                with wave.open(source["path"]) as stored:
                    self.assertEqual((stored.getframerate(), stored.getnchannels(), stored.getnframes()), (16000, 1, 24000))
            for start, end, detail in ((2.0, 2.0, "window_empty"), (5.0, 6.0, "window_empty"), (0.0, 0.01, "window_too_short")):
                with self.assertRaises(VoiceProviderUnavailable) as caught:
                    voice_sepformer.separate_file(separator, audio, directory, start=start, end=end)
                self.assertEqual((caught.exception.reason, caught.exception.detail), ("inference_failed", detail))


if __name__ == "__main__":
    unittest.main()
