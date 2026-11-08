"""ECAPA-TDNN (SpeechBrain) adapter, #356.

Two families of tests:

* Security tests always run: trapped YAML and pickles must be refused before anything is executed. They need
  PyYAML only, never torch or the model.
* Real inference tests are opt-in (``SNITCH_ECAPA_MODEL_DIR``: a local copy of the pinned
  ``speechbrain/spkrec-ecapa-voxceleb`` commit, never downloaded by the tests) and need torch, torchaudio and
  speechbrain. Without them they are skipped with an explicit reason, never simulated::

    SNITCH_ECAPA_MODEL_DIR=/path/to/model python -m unittest tests.test_voice_ecapa

Audio is synthetic (generated in a temporary directory); no clip of the real library is used.
"""
import hashlib
import importlib.util
import json
import math
import os
import pickle
import random
import shutil
import struct
import tempfile
import unittest
import wave
import zipfile
from pathlib import Path

from snitch_worker import voice, voice_ecapa
from snitch_worker.voice_ecapa import UnsupportedAudio, UnsupportedVariant

HERE = Path(__file__).parent
FIXTURES = HERE / "fixtures" / "ecapa"
ROOT_RELEASE = HERE.parent / "engine-releases" / "speechbrain-ecapa-1.0.3-r1"
REAL_YAML = (FIXTURES / "hyperparams.yaml").read_text()
REVISION = "0f99f2d0ebe89ac095bcc5903c4dd8f72b367286"
REPO = "speechbrain/spkrec-ecapa-voxceleb"
MODEL_DIR = os.environ.get("SNITCH_ECAPA_MODEL_DIR", "")
HAVE_YAML = importlib.util.find_spec("yaml") is not None
HAVE_STACK = all(importlib.util.find_spec(name) for name in ("torch", "torchaudio", "speechbrain"))
SKIP_REAL = "real inference: set SNITCH_ECAPA_MODEL_DIR to a local copy of the pinned model and install torch, torchaudio and speechbrain"


def _replace(text, old, new):
    assert old in text
    return text.replace(old, new)


@unittest.skipUnless(HAVE_YAML, "PyYAML is required")
class HyperparamsScanTest(unittest.TestCase):
    def refused(self, text, detail="hyperparams_tag_refused"):
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.check_architecture(voice_ecapa.scan_hyperparams(text))
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, detail)

    def test_the_real_file_is_accepted_without_executing_it(self):
        voice_ecapa.check_architecture(voice_ecapa.scan_hyperparams(REAL_YAML))

    def test_apply_of_a_foreign_module_is_refused(self):
        self.refused(REAL_YAML + "\nboom: !apply:os.system\n  - touch /tmp/pwned\n")

    def test_new_of_a_foreign_module_is_refused(self):
        self.refused(_replace(REAL_YAML, "!new:speechbrain.processing.features.InputNormalization", "!new:subprocess.Popen", ))

    def test_python_tags_are_refused(self):
        for tag in ("!!python/object/apply:os.system [echo hi]", "!!python/name:os.system ''", "!!python/object/new:os.system [x]"):
            self.refused(REAL_YAML + "\nboom: " + tag + "\n")

    def test_code_running_modules_inside_the_allowed_namespaces_are_refused(self):
        for dotted in ("speechbrain.utils.superpowers.run_shell", "torch.load", "torch.hub.load", "torch.jit.load",
                       "speechbrain.utils.fetching.fetch", "torch.utils.cpp_extension.load_inline"):
            self.refused(REAL_YAML + "\nboom: !apply:" + dotted + " [x]\n")

    def test_lookalike_namespaces_are_refused(self):
        for dotted in ("speechbrainx.os.system", "torchvision.io.read_file", "speechbrain", "speechbrain._private.x", "os.path.join"):
            self.refused(REAL_YAML + "\nboom: !new:" + dotted + " {}\n")

    def test_other_tags_are_refused(self):
        for tag in ("!include:other.yaml", "!cmd 'id'", "!copy <n_mels>", "!tuple [1, 2]", "!!binary aGk="):
            self.refused(REAL_YAML + "\nboom: " + tag + "\n")

    def test_ref_expressions_are_refused(self):
        for expression in ("__import__('os').system('id')", "<n_mels>.__class__", "<missing>", "<n_mels> + open('/etc/passwd')"):
            self.refused(REAL_YAML + "\nboom: !ref " + expression + "\n")

    def test_a_tagged_key_is_refused(self):
        self.refused(REAL_YAML + "\n? !apply:os.system [id]\n: 1\n")

    def test_aliases_do_not_expand_into_a_bomb(self):
        bomb = "a: &a [x, x, x, x, x, x, x, x, x]\n" + "".join(
            "%s: &%s [*%s, *%s, *%s, *%s, *%s, *%s, *%s, *%s, *%s]\n" % (n, n, p, p, p, p, p, p, p, p, p)
            for p, n in zip("abcdefghi", "bcdefghij"))
        # Nodes are shared, never expanded: the scan is linear and returns at once.
        voice_ecapa.scan_hyperparams(bomb)
        with self.assertRaises(UnsupportedVariant):
            voice_ecapa.scan_hyperparams(bomb + "x: !apply:os.system [id]\n")

    def test_unsupported_architectures_are_refused_with_a_reason(self):
        self.refused(_replace(REAL_YAML, "lin_neurons: 192", "lin_neurons: 256"), "unsupported_architecture")
        self.refused(_replace(REAL_YAML, "channels: [1024, 1024, 1024, 1024, 3072]", "channels: [512, 512, 512, 512, 1536]"), "unsupported_architecture")
        self.refused(_replace(REAL_YAML, "n_mels: 80", "n_mels: 40"), "unsupported_architecture")
        self.refused(_replace(REAL_YAML, "speechbrain.lobes.models.ECAPA_TDNN.ECAPA_TDNN", "speechbrain.lobes.models.Xvector.Xvector"), "unsupported_architecture")
        self.refused(_replace(REAL_YAML, "norm_type: sentence", "norm_type: global"), "unsupported_features")

    def test_invalid_yaml_is_refused(self):
        self.refused("a: [1, 2\n", "hyperparams_invalid")
        self.refused("- 1\n- 2\n", "hyperparams_invalid")
        self.refused("a: " + "x" * (voice_ecapa.MAX_YAML_BYTES + 1), "hyperparams_too_large")


def _trap_checkpoint(path, marker):
    """A zip checkpoint whose data.pkl runs ``os.system`` when unpickled."""
    class Trap:
        def __reduce__(self):
            return (os.system, ("touch " + str(marker),))
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("archive/data.pkl", pickle.dumps(Trap(), protocol=2))
        archive.writestr("archive/version", "3\n")


def _sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


class FilesAndCheckpointTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.dir = Path(self.tmp.name) / "model"
        self.dir.mkdir()
        self.marker = Path(self.tmp.name) / "pwned"
        (self.dir / "hyperparams.yaml").write_text(REAL_YAML)
        _trap_checkpoint(self.dir / "embedding_model.ckpt", self.marker)

    def pins(self, **overrides):
        files = {"hyperparams.yaml": _sha(self.dir / "hyperparams.yaml"), "embedding_model.ckpt": _sha(self.dir / "embedding_model.ckpt")}
        files.update(overrides)
        return {"test@pinned": {"files": files}}

    def test_unknown_files_are_refused_by_the_real_pins(self):
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir)
        self.assertEqual(caught.exception.detail, "unsupported_revision")

    def test_a_modified_file_is_refused_before_anything_is_read(self):
        pins = self.pins(**{"embedding_model.ckpt": "0" * 64})
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.load_embedder(str(self.dir), "cpu", pins)
        self.assertEqual(caught.exception.detail, "unsupported_revision")
        self.assertFalse(self.marker.exists())

    def test_a_substituted_optional_file_is_refused(self):
        (self.dir / "classifier.ckpt").write_bytes(b"swapped")
        pins = {"test@pinned": {**self.pins()["test@pinned"], "optional": {"classifier.ckpt": "1" * 64}}}
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir, pins)
        self.assertEqual(caught.exception.detail, "file_hash_mismatch")

    def test_code_files_links_and_extra_files_are_refused(self):
        pins = self.pins()
        (self.dir / "modeling.py").write_text("print('remote code')")
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir, pins)
        self.assertEqual(caught.exception.detail, "code_file_in_model")
        (self.dir / "modeling.py").unlink()
        (self.dir / "extra.bin").write_bytes(b"x")
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir, pins)
        self.assertEqual(caught.exception.detail, "unexpected_file")
        (self.dir / "extra.bin").unlink()
        (self.dir / "link.txt").symlink_to(self.dir / "hyperparams.yaml")
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir, pins)
        self.assertEqual(caught.exception.detail, "unexpected_directory_or_link")

    def test_a_missing_file_is_refused(self):
        pins = self.pins()
        (self.dir / "embedding_model.ckpt").unlink()
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.verify_files(self.dir, pins)
        self.assertEqual(caught.exception.detail, "model_file_missing")

    def test_a_malicious_checkpoint_with_a_matching_hash_is_still_refused_and_never_executed(self):
        # Even if the publisher's pin were wrong, the pickle opcodes are scanned before any load.
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.scan_checkpoint(self.dir / "embedding_model.ckpt")
        self.assertEqual(caught.exception.detail, "checkpoint_forbidden_object")
        if HAVE_YAML:
            with self.assertRaises(UnsupportedVariant) as caught:
                voice_ecapa.load_embedder(str(self.dir), "cpu", self.pins())
            self.assertEqual(caught.exception.detail, "checkpoint_forbidden_object")
        self.assertFalse(self.marker.exists(), "the trap must never run")

    def test_a_raw_legacy_pickle_is_refused_without_being_read(self):
        class Trap:
            def __reduce__(self):
                return (os.system, ("touch " + str(self.marker),))
        Trap.marker = self.marker
        (self.dir / "legacy.ckpt").write_bytes(pickle.dumps(Trap()))
        with self.assertRaises(UnsupportedVariant) as caught:
            voice_ecapa.scan_checkpoint(self.dir / "legacy.ckpt")
        self.assertEqual(caught.exception.detail, "checkpoint_unsupported_format")
        self.assertFalse(self.marker.exists())

    def test_stack_global_and_builtins_are_refused(self):
        for payload in (pickle.dumps(eval, protocol=4), pickle.dumps(os.system, protocol=4), pickle.dumps(os.system, protocol=2)):
            path = self.dir / "x.ckpt"
            with zipfile.ZipFile(path, "w") as archive:
                archive.writestr("a/data.pkl", payload)
            with self.assertRaises(UnsupportedVariant):
                voice_ecapa.scan_checkpoint(path)

    def test_a_plain_state_dict_pickle_shape_is_accepted(self):
        import collections
        payload = pickle.dumps(collections.OrderedDict(a=1), protocol=2)
        path = self.dir / "ok.ckpt"
        with zipfile.ZipFile(path, "w") as archive:
            archive.writestr("a/data.pkl", payload)
        voice_ecapa.scan_checkpoint(path)


class AudioFormatTest(unittest.TestCase):
    def test_unsupported_containers_are_refused(self):
        for name in ("voice.mp3", "voice.m4a", "voice.txt", "voice", "voice.wav.exe"):
            with self.assertRaises(UnsupportedAudio) as caught:
                voice_ecapa.check_audio_file(name)
            self.assertEqual(caught.exception.detail, "unsupported_audio_format")
            self.assertEqual(caught.exception.reason, "inference_failed")

    def test_supported_containers_are_explicit(self):
        self.assertEqual(voice_ecapa.SUPPORTED_EXTENSIONS, (".wav", ".flac", ".ogg"))
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "a.WAV")
            Path(path).write_bytes(b"x")
            voice_ecapa.check_audio_file(path)


# -- synthetic audio ---------------------------------------------------------------------------------------------

def synthetic_voice(path, f0, seconds=3.0, rate=16000, channels=1, seed=7):
    """A deterministic vowel-like signal: harmonics of ``f0`` with a formant-shaped envelope, vibrato and noise."""
    rng = random.Random(seed)
    frames = bytearray()
    for i in range(int(seconds * rate)):
        t = i / rate
        phase = 2 * math.pi * f0 * (t + 0.004 * math.sin(2 * math.pi * 5 * t))
        value = sum(math.exp(-((k * f0 - 700) / 500) ** 2) * math.sin(k * phase) / k ** 0.5 for k in range(1, 20))
        sample = int(max(-1.0, min(1.0, 0.3 * value + rng.uniform(-0.01, 0.01))) * 32000)
        frames += struct.pack("<h", sample) * channels
    with wave.open(path, "wb") as out:
        out.setnchannels(channels)
        out.setsampwidth(2)
        out.setframerate(rate)
        out.writeframes(bytes(frames))


@unittest.skipUnless(MODEL_DIR and HAVE_STACK and HAVE_YAML, SKIP_REAL)
class RealInferenceTest(unittest.TestCase):
    """The pinned model, really loaded and run on CPU. Mocks would prove nothing here."""

    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.paths = {}
        for name, f0 in (("a", 110.0), ("b", 210.0)):
            cls.paths[name] = os.path.join(cls.tmp.name, name + ".wav")
            synthetic_voice(cls.paths[name], f0)
        cls.open_audio = staticmethod(voice_ecapa.speaker_loader(MODEL_DIR, "cpu"))

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def embed(self, name="a", start=0.0, end=2.5, opener=None):
        return (opener or self.open_audio)(self.paths[name])(start, end)

    def test_the_directory_is_the_pinned_commit(self):
        self.assertEqual(voice_ecapa.verify_files(MODEL_DIR), REPO + "@" + REVISION)

    def test_dimensions_and_finiteness(self):
        vector = self.embed()
        self.assertEqual(len(vector), voice_ecapa.EMBEDDING_DIMENSIONS)
        self.assertTrue(all(math.isfinite(x) for x in vector))
        self.assertGreater(math.sqrt(sum(x * x for x in vector)), 0.0)

    def test_deterministic_within_float32_tolerance(self):
        first, second = self.embed(), self.embed()
        # Same process, same inputs, eval mode, no dropout: only float32 reduction order may differ. Cosine
        # distance must stay below 1e-5 (float32 epsilon is 1.2e-7, accumulated over ~1e4 operations).
        self.assertLess(1 - _cosine(first, second), 1e-5)
        # A freshly loaded model gives the same answer: nothing depends on load order or caches.
        again = self.embed(opener=voice_ecapa.speaker_loader(MODEL_DIR, "cpu"))
        self.assertLess(1 - _cosine(first, again), 1e-5)

    def test_it_reacts_to_the_audio_without_claiming_to_identify_speakers(self):
        a, b = self.embed("a"), self.embed("b")
        # Synthetic tones are not speech: only that the embedding is not constant is asserted.
        self.assertLess(_cosine(a, b), 0.9999)

    def test_golden_vector_of_the_pinned_weights(self):
        golden = json.loads((FIXTURES / "golden-a.json").read_text())
        self.assertEqual(golden["model"], REPO + "@" + REVISION)
        # 1e-4 cosine distance: other CPUs/BLAS and torch builds move float32 results by about 1e-6; a different
        # checkpoint or feature pipeline moves them by orders of magnitude more.
        self.assertLess(1 - _cosine(self.embed("a"), golden["embedding"]), 1e-4)

    def test_resampled_stereo_input_is_supported(self):
        path = os.path.join(self.tmp.name, "stereo48.wav")
        synthetic_voice(path, 110.0, rate=48000, channels=2)
        vector = self.open_audio(path)(0.0, 2.5)
        self.assertEqual(len(vector), voice_ecapa.EMBEDDING_DIMENSIONS)
        self.assertGreater(_cosine(vector, self.embed("a")), 0.5)

    def test_unsupported_audio_is_refused_with_a_reason(self):
        with self.assertRaises(UnsupportedAudio) as caught:
            self.open_audio(os.path.join(self.tmp.name, "voice.mp3"))
        self.assertEqual(caught.exception.detail, "unsupported_audio_format")
        odd = os.path.join(self.tmp.name, "odd.wav")
        synthetic_voice(odd, 110.0, rate=11025)
        with self.assertRaises(UnsupportedAudio) as caught:
            self.open_audio(odd)
        self.assertEqual(caught.exception.detail, "unsupported_sample_rate")
        junk = os.path.join(self.tmp.name, "junk.wav")
        Path(junk).write_bytes(os.urandom(2048))
        with self.assertRaises(UnsupportedAudio) as caught:
            self.open_audio(junk)
        self.assertEqual(caught.exception.detail, "audio_undecodable")

    def test_hyperpyyaml_is_never_used(self):
        import hyperpyyaml
        calls = []
        original = hyperpyyaml.load_hyperpyyaml
        hyperpyyaml.load_hyperpyyaml = lambda *a, **k: calls.append(1) or original(*a, **k)
        try:
            voice_ecapa.speaker_loader(MODEL_DIR, "cpu")
        finally:
            hyperpyyaml.load_hyperpyyaml = original
        self.assertEqual(calls, [])

    def test_a_tampered_copy_of_the_real_model_is_refused(self):
        with tempfile.TemporaryDirectory() as copy:
            for name in ("hyperparams.yaml", "embedding_model.ckpt"):
                shutil.copy(os.path.join(MODEL_DIR, name), copy)
            Path(copy, "hyperparams.yaml").write_text(REAL_YAML + "\nboom: !apply:os.system [id]\n")
            with self.assertRaises(UnsupportedVariant):
                voice_ecapa.load_embedder(copy, "cpu")

    def test_the_runtime_bridge_verifies_and_fingerprints_with_the_real_model(self):
        """The exact entry point the Worker runs inside an installed runtime (engine_bridge), end to end."""
        from snitch_worker import engine_bridge
        manifest = json.loads((ROOT_RELEASE / "manifest.json").read_text())
        base = {"manifest": manifest, "path": MODEL_DIR, "manifest_path": os.path.join(self.tmp.name, "voice.json")}
        self.assertEqual(engine_bridge.execute({**base, "operation": "verify"}), {})
        diarization = {"job_id": "synthetic", "speakers": [{"label": "S0", "turns": [{"index": 0, "start": 0.0, "end": 2.0}]}]}
        config = {"provider": "pyannote", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
        result = engine_bridge.execute({**base, "operation": "fingerprint", "config": config, "diarization": diarization,
                                        "audio": self.paths["a"], "stream": 0})
        self.assertEqual(result["dimensions"], voice_ecapa.EMBEDDING_DIMENSIONS)
        self.assertTrue(result["fingerprints"])

    def test_fingerprint_contract_through_the_worker_pipeline(self):
        """Model, version, dimensions, unit vectors, duration and quality, exactly as sent to Snitch (ADR 0044)."""
        diarization = {"job_id": "synthetic", "speakers": [
            {"label": "S0", "turns": [{"index": 0, "start": 0.0, "end": 2.0}, {"index": 1, "start": 2.0, "end": 2.9}]}]}
        config = {"provider": "pyannote", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
        audio = self.open_audio(self.paths["a"])
        result = voice.fingerprints_from_embedder(config, diarization, audio, 0, REPO, REVISION)
        self.assertEqual((result["embedding_model"], result["embedding_model_version"]), (REPO, REVISION))
        self.assertEqual(result["dimensions"], voice_ecapa.EMBEDDING_DIMENSIONS)
        speaker = [f for f in result["fingerprints"] if f["scope"] == "speaker"][0]
        for item in result["fingerprints"]:
            self.assertEqual(len(item["embedding"]), result["dimensions"])
            # 6-decimal rounding of 192 components: the norm error is bounded by sqrt(192) * 5e-7 < 1e-5.
            self.assertAlmostEqual(math.sqrt(sum(x * x for x in item["embedding"])), 1.0, delta=1e-5)
            self.assertGreater(item["end"], item["start"])
            self.assertTrue(0.0 <= item["quality"] <= 1.0)
        self.assertAlmostEqual(speaker["end"] - speaker["start"], 2.9, delta=1e-6)


def _cosine(a, b):
    dot = sum(x * y for x, y in zip(a, b))
    return dot / (math.sqrt(sum(x * x for x in a)) * math.sqrt(sum(y * y for y in b)))


if __name__ == "__main__":
    unittest.main()
