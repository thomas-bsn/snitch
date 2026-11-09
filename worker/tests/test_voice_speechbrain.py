"""Generic ECAPA-TDNN adapter of the on-demand SpeechBrain engine (#374).

The same fixed safeguards as the pinned ECAPA adapter, minus the variant pins: no repository code, the hyperparams
YAML is data (never HyperPyYAML), weights are scanned and read with weights_only, nothing is fetched. No torch or
SpeechBrain is needed: the library is replaced by fakes. Separation has no adapter. Synthetic values only."""
import collections
import importlib.util
import os
import pickle
import sys
import tempfile
import types
import unittest
import zipfile
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_bridge, voice_ecapa
from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_ecapa import UnsupportedVariant

FIXTURES = Path(__file__).parent / "fixtures" / "ecapa"
REAL_YAML = (FIXTURES / "hyperparams.yaml").read_text()
HAVE_YAML = importlib.util.find_spec("yaml") is not None


def write_checkpoint(path, payload=None):
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("archive/data.pkl", pickle.dumps(collections.OrderedDict(a=1) if payload is None else payload, protocol=2))
        archive.writestr("archive/version", "3\n")


class FakeLibrary:
    """Stands for torch and the SpeechBrain classes the adapter builds; records how weights were read."""

    def __init__(self):
        self.loads = []
        self.built = []
        library = self

        class Model:
            def __init__(self, **kwargs):
                library.built.append(kwargs)

            def to(self, *_):
                return self

            def eval(self):
                return self

            def load_state_dict(self, state, strict):
                library.state = (state, strict)

        torch = types.ModuleType("torch")
        torch.load = lambda path, **kwargs: library.loads.append((str(path), kwargs)) or {}
        torch.ones = lambda *a, **k: None
        modules = {"torch": torch}
        for name, attribute in (("speechbrain.lobes.models.ECAPA_TDNN", "ECAPA_TDNN"), ("speechbrain.lobes.features", "Fbank"),
                                ("speechbrain.processing.features", "InputNormalization")):
            module = types.ModuleType(name)
            setattr(module, attribute, Model)
            modules[name] = module
        for name in ("speechbrain", "speechbrain.lobes", "speechbrain.lobes.models", "speechbrain.processing"):
            modules[name] = types.ModuleType(name)
        self.modules = modules


@unittest.skipUnless(HAVE_YAML, "PyYAML is required")
class GenericLoadTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name) / "model"
        self.dir.mkdir()
        self.marker = Path(self.tmp.name) / "pwned"
        (self.dir / "hyperparams.yaml").write_text(REAL_YAML)
        write_checkpoint(self.dir / "embedding_model.ckpt")

    def refused(self, detail):
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.load_embedder(str(self.dir), "cpu", pinned=False)
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, detail)
        self.assertFalse(self.marker.exists())

    def test_a_directory_of_any_revision_is_loaded_with_weights_only_and_the_fixed_architecture(self):
        library = FakeLibrary()
        with patch.dict(sys.modules, library.modules):
            embed = voice_ecapa.load_embedder(str(self.dir), "cpu", pinned=False)
        self.assertTrue(callable(embed))
        self.assertEqual(len(library.loads), 1)
        self.assertEqual(library.loads[0][1].get("weights_only"), True)
        self.assertEqual(library.built[0], voice_ecapa.ARCH_PARAMS)

    def test_the_pinned_path_still_refuses_an_unknown_revision(self):
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.load_embedder(str(self.dir), "cpu")
        self.assertEqual(caught.exception.detail, "unsupported_revision")

    def test_repository_code_links_and_foreign_weight_formats_are_refused(self):
        (self.dir / "modeling.py").write_text("print('remote code')")
        self.refused("code_file_in_model")
        (self.dir / "modeling.py").unlink()
        (self.dir / "weights.pt").write_bytes(b"x")
        self.refused("unexpected_file")
        (self.dir / "weights.pt").unlink()
        (self.dir / "link.ckpt").symlink_to(self.dir / "embedding_model.ckpt")
        self.refused("unexpected_directory_or_link")

    def test_a_missing_checkpoint_is_refused(self):
        (self.dir / "embedding_model.ckpt").unlink()
        self.refused("model_file_missing")

    def test_a_hyperparams_program_is_read_as_data_and_refused(self):
        (self.dir / "hyperparams.yaml").write_text(REAL_YAML + "\nboom: !apply:os.system [touch " + str(self.marker) + "]\n")
        self.refused("hyperparams_tag_refused")
        (self.dir / "hyperparams.yaml").write_text(REAL_YAML + "\nboom: !!python/object/apply:os.system [touch " + str(self.marker) + "]\n")
        self.refused("hyperparams_tag_refused")

    def test_another_architecture_is_refused(self):
        (self.dir / "hyperparams.yaml").write_text(REAL_YAML.replace("lin_neurons: 192", "lin_neurons: 256"))
        self.refused("unsupported_architecture")

    def test_a_malicious_checkpoint_is_refused_and_never_executed(self):
        class Trap:
            def __reduce__(self):
                return (os.system, ("touch " + str(self_marker),))
        self_marker = self.marker
        write_checkpoint(self.dir / "embedding_model.ckpt", Trap())
        library = FakeLibrary()
        with patch.dict(sys.modules, library.modules):
            self.refused("checkpoint_forbidden_object")
        self.assertEqual(library.loads, [])


class BridgeTests(unittest.TestCase):
    def request(self, kind, operation="verify"):
        runtime = {"source": "on_demand", "family": "speechbrain", "engine": "speechbrain", "distribution": "speechbrain",
                   "version": "1.0.3", "kind": kind, "kinds": ["fingerprint", "separation"], "python": "3.12", "platform": "linux-x86_64",
                   "cuda": "", "disk_bytes": 1, "vram_bytes": 0, "ram_bytes": 0, "estimated_seconds": 1}
        return {"manifest": runtime, "path": "/nonexistent", "manifest_path": "/nonexistent/voice.json", "operation": operation}

    def test_separation_has_no_adapter_and_is_refused_before_anything_is_loaded(self):
        with patch.object(voice_ecapa, "load_embedder") as load, self.assertRaises(VoiceProviderUnavailable) as caught:
            engine_bridge.execute(self.request("separation"))
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, "separation_not_available")
        load.assert_not_called()

    def test_the_bridge_reports_the_refusal_code_of_separation_to_the_worker(self):
        with tempfile.TemporaryDirectory() as directory:
            incoming, outgoing = Path(directory) / "in.json", Path(directory) / "out.json"
            import json
            incoming.write_text(json.dumps(self.request("separation")))
            saved, sys.argv = sys.argv, ["engine_bridge", str(incoming), str(outgoing)]
            try:
                engine_bridge.main()
            finally:
                sys.argv = saved
            self.assertEqual(json.loads(outgoing.read_text()), {"error": "model_load_failed", "detail": "separation_not_available"})

    def test_a_fingerprint_model_of_the_family_loads_through_the_generic_adapter(self):
        loaded = []

        def loader(path, device, variants=None, pinned=True):
            loaded.append((path, pinned))
            raise VoiceProviderUnavailable("model_load_failed")

        with patch.object(voice_ecapa, "speaker_loader", side_effect=lambda path, device, pinned=True: loader(path, device, pinned=pinned)):
            with self.assertRaises(VoiceProviderUnavailable):
                engine_bridge._speaker_loader("speechbrain", "/model", "cpu")
        self.assertEqual(loaded, [("/model", False)])

    def test_the_signed_ecapa_engine_keeps_its_pinned_loader(self):
        calls = []
        with patch.object(voice_ecapa, "speaker_loader", side_effect=lambda *a, **k: calls.append((a, k)) or "loader"):
            self.assertEqual(engine_bridge._speaker_loader("speechbrain-ecapa", "/model", "cpu"), "loader")
        self.assertEqual(calls, [(("/model", "cpu"), {})])


if __name__ == "__main__":
    unittest.main()
