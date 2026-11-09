"""A model ordered on an on-demand family runs end to end on the Worker (#372, ADR 0056): the order is accepted whatever
adapter name the catalogue gives the model, the engine installs from the local wheelhouse, the binding written after
the test inference validates, the runtime provider accepts it at job time and the bridge dispatches on the runtime's
family and kind. SpeechBrain separation and the signed and embedded contracts keep their rules. Synthetic values only."""

import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import engine_bridge, engine_runtime as runtime_module, model_orders, voice_runtime
from snitch_worker.voice import VoiceProviderUnavailable

import wheelhouse

ROOT = None
HOUSES = {}

# (family, kind, adapter name the catalogue gives such a model)
PAIRS = [("nemo", "diarization", "nemo"), ("nemo", "fingerprint", "nemo-titanet"),
         ("speechbrain", "fingerprint", "speechbrain-ecapa"), ("wespeaker", "fingerprint", "wespeaker")]
BUILDERS = {"nemo": wheelhouse.build_nemo_wheelhouse, "speechbrain": wheelhouse.build_speechbrain_wheelhouse,
            "wespeaker": wheelhouse.build_wespeaker_wheelhouse}
VERSIONS = {"nemo": "3.0.0", "speechbrain": "1.0.3", "wespeaker": "1.0.1"}


def setUpModule():
    global ROOT
    ROOT = tempfile.TemporaryDirectory()
    for family, build in BUILDERS.items():
        HOUSES[family] = build(Path(ROOT.name) / family)


def tearDownModule():
    ROOT.cleanup()


def runtime(family, kind):
    f = runtime_module.FAMILIES[family]
    return {"source": "on_demand", "family": family, "engine": f["engine"], "distribution": f["distribution"],
            "version": VERSIONS[family], "kind": kind, "kinds": list(f["kinds"]), "python": "3.12", "platform": "linux-x86_64",
            "cuda": "", "disk_bytes": 1 << 20, "vram_bytes": 0, "ram_bytes": 1 << 20, "estimated_seconds": 60}


def model_order(engine, kind, rt, **changes):
    order = {"action": "install", "kind": kind, "engine": engine, "model_id": "synthetic-family-model", "repo": "synthetic/family-model",
             "revision": "0" * 40, "idempotency_key": "family-model-0001", "runtime": rt, "runtime_intent": "install"}
    order.update(changes)
    return order


class Legacy:
    """The embedded provider the runtime provider wraps: it only locates the model and names its environment."""

    def __init__(self, path, environ, engine):
        self.path, self._environ, self.engine = path, environ, engine

    def _embedding_location(self, config):
        return self.path

    def _diarization_path(self, config):
        return self.path


class FamilyModelTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.model = os.path.join(self.directory.name, "model")
        os.makedirs(self.model)
        self.env = {"SNITCH_WORKER_MODELS_DIR": os.path.join(self.directory.name, "models"), "PATH": os.environ.get("PATH", ""),
                    "HOME": self.directory.name, "SNITCH_ENGINE_NO_INDEX": "1"}
        for p in (patch.object(runtime_module.sys, "version_info", (3, 12)), patch.object(runtime_module.platform, "system", return_value="Linux"),
                  patch.object(runtime_module.platform, "machine", return_value="x86_64")):
            p.start()
            self.addCleanup(p.stop)

    def installed(self, family, kind):
        rt = runtime(family, kind)
        runtime_module.install(rt, {**self.env, "SNITCH_ENGINE_FIND_LINKS": str(HOUSES[family])}, lambda *_: None, True)
        self.assertTrue(runtime_module.installed(rt, self.env))
        return rt


class OrderContractTest(FamilyModelTest):
    def test_every_active_pair_is_accepted_whatever_the_adapter_name(self):
        for family, kind, adapter in PAIRS:
            with self.subTest(family=family, kind=kind):
                self.assertTrue(model_orders.OrderExecutor._well_formed(model_order(adapter, kind, runtime(family, kind))))

    def test_speechbrain_separation_installs_an_engine_but_never_a_model(self):
        self.assertFalse(model_orders.OrderExecutor._well_formed(model_order("speechbrain-ecapa", "separation", runtime("speechbrain", "separation"))))

    def test_a_runtime_of_another_kind_or_a_kind_the_family_lacks_is_refused(self):
        self.assertFalse(model_orders.OrderExecutor._well_formed(model_order("nemo", "diarization", runtime("nemo", "fingerprint"))))
        self.assertFalse(model_orders.OrderExecutor._well_formed(model_order("wespeaker", "diarization", runtime("wespeaker", "fingerprint"))))

    def test_the_signed_and_embedded_contracts_are_unchanged(self):
        well = model_orders.OrderExecutor._well_formed
        self.assertTrue(well(model_order("nemo", "diarization", None)))
        self.assertTrue(well(model_order("pyannote", "fingerprint", None)))
        # A signed adapter still needs its release, and a family adapter name without a runtime is refused.
        self.assertFalse(well(model_order("nemo-titanet", "fingerprint", None)))
        self.assertFalse(well(model_order("speechbrain-ecapa", "fingerprint", None)))
        self.assertFalse(well(model_order("wespeaker", "fingerprint", None)))
        self.assertFalse(well(model_order("unknown-engine", "fingerprint", None)))


class EndToEndTest(FamilyModelTest):
    def test_each_family_installs_binds_and_is_accepted_at_job_time(self):
        for family, kind, adapter in PAIRS:
            with self.subTest(family=family, kind=kind):
                rt = self.installed(family, kind)
                runtime_module.bind(self.model, rt)
                self.assertEqual(runtime_module.model_binding(self.model), rt)
                provider = voice_runtime.RuntimeProvider(Legacy(self.model, self.env, adapter))
                config = {"fingerprint_model": {"engine": adapter}} if kind == "fingerprint" else {"provider": adapter}
                path, manifest = provider._binding(config, kind)
                self.assertEqual((path, manifest), (self.model, rt))

    def test_a_binding_of_another_kind_is_refused_at_job_time(self):
        rt = self.installed("nemo", "diarization")
        runtime_module.bind(self.model, rt)
        provider = voice_runtime.RuntimeProvider(Legacy(self.model, self.env, "nemo-titanet"))
        with self.assertRaises(VoiceProviderUnavailable):
            provider._binding({"fingerprint_model": {"engine": "nemo-titanet"}}, "fingerprint")

    def test_the_bridge_verifies_each_family_with_the_adapter_of_its_family_and_kind(self):
        calls = []
        for family, kind, adapter in PAIRS:
            with self.subTest(family=family, kind=kind):
                calls.clear()
                rt = self.installed(family, kind)
                request = {"manifest": rt, "path": self.model, "manifest_path": os.path.join(self.directory.name, "voice.json"), "operation": "verify"}

                class Provider:
                    def __init__(self, label):
                        self.label = label

                def speaker(engine, **kwargs):
                    calls.append(("speaker", engine))
                    return Provider(engine)

                def nemo_family(manifest, env):
                    calls.append(("nemo-family", manifest["kind"]))
                    return Provider("nemo")

                with patch.object(engine_bridge, "SpeakerProvider", side_effect=speaker), \
                        patch("snitch_worker.voice_nemo_family.provider", side_effect=nemo_family), \
                        patch("snitch_worker.voice_selftest.embed_synthetic", side_effect=lambda p: calls.append(("embed", p.label))), \
                        patch("snitch_worker.voice_selftest.diarize_synthetic", side_effect=lambda p, e: calls.append(("diarize", e))):
                    self.assertEqual(engine_bridge.execute(request), {})
                expected = {("nemo", "diarization"): [("nemo-family", "diarization"), ("diarize", "nemo")],
                            ("nemo", "fingerprint"): [("nemo-family", "fingerprint"), ("embed", "nemo")],
                            ("speechbrain", "fingerprint"): [("speaker", "speechbrain"), ("embed", "speechbrain")],
                            ("wespeaker", "fingerprint"): [("speaker", "wespeaker"), ("embed", "wespeaker")]}[(family, kind)]
                self.assertEqual(calls, expected)


if __name__ == "__main__":
    unittest.main()
