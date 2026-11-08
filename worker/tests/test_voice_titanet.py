"""TitaNet adapter (#355).

Two kinds of tests, never mixed up:

* Format and refusal tests run everywhere. They need neither torch nor NeMo: they feed synthetic ``.nemo``
  archives and configs to the pure-Python validation and check what is refused, and why.
* Real-inference tests load the real public model and run it on this machine's CPU. They are opt-in because
  they need the ``voice`` extra and the weights (about 100 MB), given by path so that nothing is downloaded
  by the tests::

      SNITCH_TITANET_TEST_PATH=/path/to/nvidia__speakerverification_en_titanet_large \\
          uv run --frozen python -m unittest tests.test_voice_titanet

  They are skipped, with this reason, when the path or the libraries are missing. No mock stands in for the
  loader or the model there. They prove nothing about a GPU: that proof is made by the operator (docs/worker-engines-326.md).

Audio is synthetic (generated WAV files in a temporary directory): no clip of any library is used.
"""

import importlib.util
import io
import json
import math
import os
import pickle
import struct
import subprocess
import sys
import tarfile
import tempfile
import unittest
import wave
from pathlib import Path

from snitch_worker import voice, voice_models, voice_titanet
from snitch_worker.voice_titanet import TitaNetRefused

REPO = "nvidia/speakerverification_en_titanet_large"
REVISION = "0dc382f40121a5fbd34db10a2bb04d826c2be6a8"
MODEL_PATH = os.environ.get("SNITCH_TITANET_TEST_PATH", "")
HAVE_STACK = bool(MODEL_PATH) and all(importlib.util.find_spec(name) for name in ("torch", "torchaudio", "nemo", "yaml", "omegaconf"))
SKIP_REASON = "set SNITCH_TITANET_TEST_PATH to a TitaNet-L model directory and install the voice extra (NeMo, torch, torchaudio)"
HAVE_YAML = importlib.util.find_spec("yaml") is not None


def block(kernel=3, filters=1024, repeat=1):
    return {"filters": filters, "repeat": repeat, "kernel": [kernel], "stride": [1], "dilation": [1], "dropout": 0.0,
            "residual": False, "separable": True, "se": True, "se_context_size": -1}


def titanet_config(**changes):
    """The shape of the published TitaNet-L configuration (reduced to what the loader reads)."""
    config = {
        "preprocessor": {"_target_": voice_titanet.PREPROCESSOR, "normalize": "per_feature", "window_size": 0.025, "sample_rate": 16000,
                         "window_stride": 0.01, "window": "hann", "features": 80, "n_fft": 512, "frame_splicing": 1, "dither": 1e-5},
        "encoder": {"_target_": voice_titanet.ENCODER, "feat_in": 80, "activation": "relu", "conv_mask": True, "jasper": [block(3), block(7, repeat=3)]},
        "decoder": {"_target_": voice_titanet.DECODER, "feat_in": 3072, "num_classes": 16681, "pool_mode": "attention", "emb_sizes": 192},
        "loss": {"_target_": "nemo.collections.asr.losses.angularloss.AngularSoftmaxLoss", "scale": 30, "margin": 0.2},
        "train_ds": {"manifest_filepath": "/manifests/train.json"},
    }
    for path, value in changes.items():
        target = config
        *parents, last = path.split(".")
        for name in parents:
            target = target[name]
        if value is None:
            target.pop(last, None)
        else:
            target[last] = value
    return config


def write_archive(path, members, kind="w"):
    """A tar file with ``members``: name -> bytes, or a TarInfo (link, device...) when a test needs one."""
    with tarfile.open(path, kind) as archive:
        for name, content in members.items():
            if isinstance(content, tarfile.TarInfo):
                content.name = name
                archive.addfile(content)
                continue
            info = tarfile.TarInfo(name)
            info.size = len(content)
            archive.addfile(info, io.BytesIO(content))


class ConfigRefusalTests(unittest.TestCase):
    def refused(self, config, detail):
        with self.assertRaises(TitaNetRefused) as caught:
            voice_titanet.validate_config(config)
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(caught.exception.detail, detail)
        self.assertNotIn("/", str(caught.exception), "the message never carries a path or a library text")

    def test_the_supported_configuration_is_accepted_without_its_training_sections(self):
        approved = voice_titanet.validate_config(titanet_config())
        self.assertEqual(approved["dimensions"], 192)
        self.assertTrue(approved["decoder"]["angular"], "AngularSoftmaxLoss is read as a name: the decoder has no bias")
        self.assertNotIn("_target_", approved["preprocessor"])
        self.assertNotIn("train_ds", approved)
        self.assertFalse(voice_titanet.validate_config(titanet_config(loss=None))["decoder"]["angular"])

    def test_another_architecture_is_an_unsupported_variant(self):
        for path, target in [("encoder._target_", "nemo.collections.asr.modules.JasperEncoder"),
                             ("decoder._target_", "nemo.collections.asr.modules.ConvASRDecoderClassification"),
                             ("preprocessor._target_", "nemo.collections.asr.modules.AudioToMFCCPreprocessor")]:
            with self.subTest(path=path):
                self.refused(titanet_config(**{path: target}), "variant_unsupported")

    def test_a_target_outside_nemo_is_executable_configuration(self):
        for path, target in [("encoder._target_", "os.system"), ("decoder._target_", "evil.module.Decoder"), ("preprocessor._target_", None)]:
            with self.subTest(path=path):
                config = titanet_config()
                section, _ = path.split(".")
                config[section]["_target_"] = target
                self.refused(config, "config_executable")

    def test_interpolations_nested_targets_and_objects_are_refused(self):
        self.refused(titanet_config(**{"encoder.activation": "${oc.env:HOME}"}), "config_executable")
        self.refused(titanet_config(**{"decoder.pool_mode": "attention", "decoder.extra": {"_target_": "os.system"}}), "config_executable")
        self.refused(titanet_config(**{"preprocessor.window": object()}), "config_executable")
        self.refused(titanet_config(**{"encoder.jasper": [dict(block(), _target_="os.system")]}), "config_executable")

    def test_parameters_that_change_the_contract_are_refused(self):
        self.refused(titanet_config(**{"preprocessor.sample_rate": 8000}), "sample_rate_unsupported")
        self.refused(titanet_config(**{"preprocessor.features": 64, "encoder.feat_in": 64}), "variant_unsupported")
        self.refused(titanet_config(**{"decoder.pool_mode": "mean"}), "variant_unsupported")
        self.refused(titanet_config(**{"decoder.emb_sizes": [256, 192]}), "variant_unsupported")
        self.refused(titanet_config(**{"decoder.emb_sizes": True}), "variant_unsupported")
        self.refused(titanet_config(**{"encoder.quantize": True}), "variant_unsupported")
        self.refused(titanet_config(**{"loss._target_": "nemo.collections.asr.losses.Other"}), "variant_unsupported")
        self.refused(titanet_config(**{"encoder.jasper": []}), "variant_unsupported")
        self.refused(titanet_config(**{"encoder.jasper": [dict(block(), hook="x")]}), "variant_unsupported")

    def test_a_missing_or_malformed_section_is_refused(self):
        for section in ("preprocessor", "encoder", "decoder"):
            with self.subTest(section=section):
                config = titanet_config()
                del config[section]
                self.refused(config, "variant_unsupported")
        self.refused([], "config_unreadable")

    @unittest.skipUnless(HAVE_YAML, "PyYAML is not installed")
    def test_yaml_is_read_as_data_only(self):
        self.assertEqual(voice_titanet.parse_config(b"a: 1\nb: [x, y]\n"), {"a": 1, "b": ["x", "y"]})
        for raw in (b"!!python/object/apply:os.system ['echo hi']", b"a: [unclosed", b"- just\n- a list\n", b"\xff\xfe"):
            with self.subTest(raw=raw):
                with self.assertRaises(TitaNetRefused) as caught:
                    voice_titanet.parse_config(raw)
                self.assertEqual(caught.exception.detail, "config_unreadable")


class ArchiveRefusalTests(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.dir = self._tmp.name
        self.addCleanup(self._tmp.cleanup)

    def refused(self, detail, path=None):
        with self.assertRaises(TitaNetRefused) as caught:
            voice_titanet.read_archive(path or self.dir)
        self.assertEqual(caught.exception.detail, detail)

    def archive(self, members, name="titanet.nemo", kind="w"):
        write_archive(os.path.join(self.dir, name), members, kind)

    def test_the_supported_format_is_a_nemo_tar_with_a_config_and_weights(self):
        for kind in ("w", "w:gz"):
            with self.subTest(kind=kind):
                self.archive({"./model_config.yaml": b"config", "./model_weights.ckpt": b"weights"}, kind=kind)
                self.assertEqual(voice_titanet.read_archive(self.dir), (b"config", b"weights"))
                self.assertEqual(voice_titanet.read_archive(os.path.join(self.dir, "titanet.nemo")), (b"config", b"weights"))

    def test_no_archive_or_two_archives_are_refused(self):
        self.refused("archive_missing")
        self.refused("archive_missing", os.path.join(self.dir, "absent"))
        (Path(self.dir) / "model.safetensors").write_bytes(b"x")
        (Path(self.dir) / "model.ckpt").write_bytes(b"x")
        self.refused("archive_missing")
        self.archive({"model_config.yaml": b"c", "model_weights.ckpt": b"w"}, "a.nemo")
        self.archive({"model_config.yaml": b"c", "model_weights.ckpt": b"w"}, "b.nemo")
        self.refused("archive_ambiguous")

    def test_a_file_that_is_not_a_tar_is_refused(self):
        (Path(self.dir) / "titanet.nemo").write_bytes(b"PK\x03\x04 this is not a tar archive at all" * 40)
        self.refused("archive_not_nemo")
        (Path(self.dir) / "titanet.nemo").write_bytes(b"")
        self.refused("archive_not_nemo")

    def test_a_truncated_archive_is_refused(self):
        self.archive({"model_config.yaml": b"c" * 4000, "model_weights.ckpt": b"w" * 4000})
        full = (Path(self.dir) / "titanet.nemo").read_bytes()
        (Path(self.dir) / "titanet.nemo").write_bytes(full[:3000])
        with self.assertRaises(TitaNetRefused):
            voice_titanet.read_archive(self.dir)

    def test_missing_members_are_named(self):
        self.archive({"model_weights.ckpt": b"w"})
        self.refused("config_missing")
        self.archive({"model_config.yaml": b"c"})
        self.refused("weights_missing")

    def test_links_escaping_paths_and_archive_bombs_are_refused(self):
        link = tarfile.TarInfo()
        link.type, link.linkname = tarfile.SYMTYPE, "/etc/passwd"
        self.archive({"model_config.yaml": b"c", "model_weights.ckpt": b"w", "link": link})
        self.refused("archive_unsafe")
        self.archive({"../escape": b"x", "model_config.yaml": b"c", "model_weights.ckpt": b"w"})
        self.refused("archive_unsafe")
        self.archive({"/absolute": b"x", "model_config.yaml": b"c", "model_weights.ckpt": b"w"})
        self.refused("archive_unsafe")
        self.archive({f"extra{i}": b"x" for i in range(voice_titanet.MAX_MEMBERS + 1)})
        self.refused("archive_unsafe")
        self.archive({"a/model_config.yaml": b"c", "b/model_config.yaml": b"c", "model_weights.ckpt": b"w"})
        self.refused("archive_unsafe")

    def test_a_repository_with_other_files_still_uses_only_the_archive(self):
        self.archive({"model_config.yaml": b"c", "model_weights.ckpt": b"w"})
        for name in ("config.json", "modeling.py", "README.md", "sample.wav"):
            (Path(self.dir) / name).write_text("print('never imported')")
        self.assertEqual(voice_titanet.read_archive(self.dir), (b"c", b"w"))


class AdapterRegistrationTests(unittest.TestCase):
    def test_the_bridge_runs_the_titanet_adapter_and_reports_the_refusal_code(self):
        from snitch_worker import engine_bridge

        with tempfile.TemporaryDirectory() as root:
            request = {"manifest": {"engine": "nemo-titanet", "kind": "fingerprint"}, "path": root, "manifest_path": os.path.join(root, "m.json"),
                       "operation": "verify"}
            incoming, outgoing = os.path.join(root, "in.json"), os.path.join(root, "out.json")
            Path(incoming).write_text(json.dumps(request))
            if importlib.util.find_spec("torch") is None or importlib.util.find_spec("torchaudio") is None:
                self.skipTest("torch and torchaudio are not installed: the bridge cannot reach the loader")
            saved, sys.argv = sys.argv, ["engine_bridge", incoming, outgoing]
            try:
                engine_bridge.main()
            finally:
                sys.argv = saved
            answer = json.loads(Path(outgoing).read_text())
        self.assertEqual(answer["error"], "model_load_failed")
        self.assertEqual(answer["detail"], "archive_missing")


class RefusalReachesTheWorkerLogTests(unittest.TestCase):
    """The answer file of the runtime sub-process carries the adapter's refusal code up to the model verification log."""

    def call(self, answer):
        from unittest.mock import patch
        from snitch_worker import engine_runtime, voice_runtime

        manifest = {"engine": "nemo-titanet", "kind": "fingerprint"}

        def run(command, **_):
            Path(command[-1]).write_text(json.dumps(answer))

        with patch.object(engine_runtime, "validate"), patch.object(engine_runtime, "installed", return_value=True), \
                patch.object(voice_runtime, "inference_device", return_value="cpu"), patch.object(voice_runtime.cancellation, "run_process", run):
            with tempfile.TemporaryDirectory() as root:
                voice_runtime.call(manifest, root, {"SNITCH_WORKER_MODELS_DIR": root}, operation="verify")

    def test_the_stable_refusal_detail_is_kept_and_a_free_text_one_is_dropped(self):
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            self.call({"error": "model_load_failed", "detail": "variant_unsupported"})
        self.assertEqual((caught.exception.reason, caught.exception.detail), ("model_load_failed", "variant_unsupported"))
        self.assertEqual(voice_models._cause_of_verification(caught.exception), "runtime_incompatible")
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            self.call({"error": "model_load_failed", "detail": "/secret/path hf_token_value"})
        self.assertFalse(hasattr(caught.exception, "detail"))


def write_wav(path, samples, rate=16000, channels=1):
    with wave.open(path, "wb") as out:
        out.setnchannels(channels)
        out.setsampwidth(2)
        out.setframerate(rate)
        out.writeframes(b"".join(struct.pack("<h", max(-32768, min(32767, int(s * 32767)))) * channels for s in samples))


@unittest.skipUnless(HAVE_STACK, SKIP_REASON)
class RealTitaNetInferenceTests(unittest.TestCase):
    """The real model, on the CPU of this machine (never a GPU claim), against synthetic audio."""

    @classmethod
    def setUpClass(cls):
        import torch

        cls.torch = torch
        torch.set_num_threads(min(4, os.cpu_count() or 1))
        cls.model = voice_titanet.load(MODEL_PATH, "cpu")
        cls.archive = voice_titanet.archive_of(MODEL_PATH)

    def wave(self, seed, seconds=4.0):
        return self.torch.tensor([voice_titanet.synthetic_waveform(seconds, seed)], dtype=self.torch.float32)

    def unit(self, seed, seconds=4.0):
        vector = self.model.embed(self.wave(seed, seconds)).reshape(-1)
        return vector / vector.norm()

    def test_identity_and_dimensions_are_those_of_titanet_large(self):
        vector = self.model.embed(self.wave(1))
        self.assertEqual(tuple(vector.shape), (1, 192))
        self.assertEqual(self.model.dimensions, 192)
        self.assertEqual(len(self.model.archive_sha256), 64)
        self.assertEqual(self.model.archive_sha256, __import__("hashlib").sha256(Path(self.archive).read_bytes()).hexdigest())

    def test_the_vector_is_finite_non_zero_and_normalizes_to_one(self):
        vector = self.model.embed(self.wave(2)).reshape(-1)
        self.assertTrue(bool(self.torch.isfinite(vector).all()))
        self.assertGreater(float(vector.norm()), 1e-3)
        self.assertAlmostEqual(float((vector / vector.norm()).norm()), 1.0, places=6)

    def test_the_dimension_does_not_depend_on_the_window(self):
        for seconds in (0.5, 1.0, 2.5, 9.0):
            with self.subTest(seconds=seconds):
                self.assertEqual(self.model.embed(self.wave(1, seconds)).shape[-1], 192)

    def test_repeated_inference_is_deterministic(self):
        """Same machine, same threads: eval mode adds no dither and no random op, so the floats are identical.
        1e-5 is the bound if a BLAS kernel were to change its reduction order (float32 epsilon is 1.2e-7 and the
        error grows with depth, so this leaves two orders of magnitude)."""
        first, second = self.unit(1), self.unit(1)
        self.assertLess(float((first - second).abs().max()), 1e-5)

    def test_the_thread_count_does_not_change_the_result_beyond_rounding(self):
        """Thread counts split the same sums differently: float32 rounding only. 1e-4 per component of a unit vector
        (smallest meaningful change for cosine matching is orders above) and the cosine stays above 1 - 1e-6."""
        previous = self.torch.get_num_threads()
        try:
            self.torch.set_num_threads(1)
            single = self.unit(3)
            self.torch.set_num_threads(max(2, previous))
            several = self.unit(3)
        finally:
            self.torch.set_num_threads(previous)
        self.assertLess(float((single - several).abs().max()), 1e-4)
        self.assertGreater(float(single @ several), 1 - 1e-6)

    def test_the_result_matches_nemos_own_loader_on_the_same_weights(self):
        """The reference is NeMo's own ``restore_from`` (trusted test-side only: it is what the adapter avoids in production).
        Our modules must reproduce it; same tolerance as the thread-count case."""
        from nemo.collections.asr.models import EncDecSpeakerLabelModel

        reference = EncDecSpeakerLabelModel.restore_from(self.archive, map_location="cpu").eval()
        signal = self.wave(1)
        with self.torch.inference_mode():
            _, expected = reference(input_signal=signal, input_signal_length=self.torch.tensor([signal.shape[-1]]))
        expected = expected.reshape(-1)
        expected = expected / expected.norm()
        actual = self.unit(1)
        self.assertLess(float((expected - actual).abs().max()), 1e-4)
        self.assertGreater(float(expected @ actual), 1 - 1e-6)

    def test_the_embedding_reacts_to_the_signal(self):
        """Not a speaker-verification claim (the audio is synthetic): the same synthetic voice with other noise is
        closer than another pitch, so the vector is not constant and carries the signal. Margins are wide
        (measured on this model: about 0.98 versus 0.28 to 0.69)."""
        reference = self.unit(1)
        same_pitch_other_noise = self.unit(6)  # seeds 1 and 6 share a pitch (110 + 20 * (seed % 5) Hz)
        other_pitch = self.unit(4)
        self.assertGreater(float(reference @ same_pitch_other_noise), 0.9)
        self.assertLess(float(reference @ other_pitch), 0.8)

    def test_inputs_the_model_cannot_embed_are_refused_not_zeroed(self):
        for waveform in (self.torch.zeros(1, 0), self.torch.full((1, 16000), float("nan")), self.torch.ones(2, 16000), self.torch.ones(16000)):
            with self.subTest(shape=tuple(waveform.shape)):
                with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                    self.model.embed(waveform)
                self.assertEqual(caught.exception.reason, "inference_failed")

    def test_a_tampered_checkpoint_is_refused_and_executes_nothing(self):
        with tempfile.TemporaryDirectory() as root:
            marker = os.path.join(root, "executed")

            class Payload:
                def __reduce__(self):
                    return (open, (marker, "w"))

            config, _ = voice_titanet.read_archive(MODEL_PATH)
            write_archive(os.path.join(root, "evil.nemo"), {"model_config.yaml": config, "model_weights.ckpt": pickle.dumps({"a": Payload()})})
            with self.assertRaises(TitaNetRefused) as caught:
                voice_titanet.load(root, "cpu")
            self.assertEqual(caught.exception.detail, "weights_incompatible")
            self.assertFalse(os.path.exists(marker), "the pickle must never run")
            write_archive(os.path.join(root, "evil.nemo"), {"model_config.yaml": config, "model_weights.ckpt": b"not a checkpoint"})
            with self.assertRaises(TitaNetRefused) as caught:
                voice_titanet.load(root, "cpu")
            self.assertEqual(caught.exception.detail, "weights_incompatible")

    def test_weights_of_another_shape_are_refused(self):
        torch = self.torch
        with tempfile.TemporaryDirectory() as root:
            config, _ = voice_titanet.read_archive(MODEL_PATH)
            buffer = io.BytesIO()
            torch.save({"decoder.final.weight": torch.zeros(3, 3)}, buffer)
            write_archive(os.path.join(root, "m.nemo"), {"model_config.yaml": config, "model_weights.ckpt": buffer.getvalue()})
            with self.assertRaises(TitaNetRefused) as caught:
                voice_titanet.load(root, "cpu")
            self.assertEqual(caught.exception.detail, "weights_incompatible")

    def test_an_executable_config_in_a_real_archive_is_refused_before_anything_loads(self):
        import yaml

        with tempfile.TemporaryDirectory() as root:
            config, weights = voice_titanet.read_archive(MODEL_PATH)
            data = yaml.safe_load(config)
            data["encoder"]["_target_"] = "os.system"
            write_archive(os.path.join(root, "m.nemo"), {"model_config.yaml": yaml.safe_dump(data).encode(), "model_weights.ckpt": weights})
            with self.assertRaises(TitaNetRefused) as caught:
                voice_titanet.load(root, "cpu")
            self.assertEqual(caught.exception.detail, "config_executable")


@unittest.skipUnless(HAVE_STACK, SKIP_REASON)
class RealTitaNetFingerprintTests(unittest.TestCase):
    """The production path: catalogue layout, approved adapter, audio decoding, the fingerprint contract (ADR 0044)."""

    @classmethod
    def setUpClass(cls):
        cls._tmp = tempfile.TemporaryDirectory()
        root = cls.root = cls._tmp.name
        cls.env = {"SNITCH_WORKER_MODELS_DIR": root, "HF_HUB_OFFLINE": "1"}
        cls.path = voice_models.model_path(cls.env, REPO, REVISION)
        os.makedirs(cls.path)
        archive = voice_titanet.archive_of(MODEL_PATH)
        os.symlink(archive, os.path.join(cls.path, os.path.basename(archive)))
        Path(cls.path, voice_models.READY_MARKER).write_text(REVISION)
        cls.mono = os.path.join(root, "mono16k.wav")
        write_wav(cls.mono, voice_titanet.synthetic_waveform(12.0, 1))
        cls.stereo = os.path.join(root, "stereo44k.wav")
        write_wav(cls.stereo, voice_titanet.synthetic_waveform(12.0, 1, rate=44100), rate=44100, channels=2)
        cls.config = {"provider": "pyannote", "model": "pyannote/speaker-diarization-community-1", "model_version": "community-1",
                      "min_fingerprint_seconds": 1.0, "min_quality": 0.4,
                      "fingerprint_model": {"engine": "nemo-titanet", "repo": REPO, "revision": REVISION}}
        cls.diarization = {"job_id": 9, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 4.0}, {"index": 1, "start": 5.0, "end": 9.0}]}]}

    @classmethod
    def tearDownClass(cls):
        cls._tmp.cleanup()

    def provider(self):
        from snitch_worker import engine_bridge

        return engine_bridge.SpeakerProvider("nemo-titanet", environ=dict(self.env), device_probe=lambda: "cpu")

    def check_contract(self, result, stream=0):
        self.assertEqual(result["embedding_model"], REPO)
        self.assertEqual(result["embedding_model_version"], REVISION)
        self.assertEqual(result["dimensions"], 192)
        self.assertEqual(result["audio_stream_index"], stream)
        self.assertEqual(result["diarization_job_id"], 9)
        scopes = [item["scope"] for item in result["fingerprints"]]
        self.assertEqual(scopes, ["speaker", "turn", "turn"])
        for item in result["fingerprints"]:
            vector = item["embedding"]
            self.assertEqual(len(vector), 192)
            self.assertAlmostEqual(math.sqrt(sum(x * x for x in vector)), 1.0, places=5)
            self.assertTrue(all(math.isfinite(x) for x in vector))
            self.assertGreater(item["end"], item["start"], "the duration of the speech the vector was computed on")
            self.assertTrue(0.0 < item["quality"] <= 1.0)
        speaker = result["fingerprints"][0]
        self.assertAlmostEqual(speaker["end"] - speaker["start"], 8.0, places=3)

    def test_a_fingerprint_carries_model_version_dimensions_unit_vectors_duration_and_quality(self):
        self.check_contract(self.provider().fingerprint(self.config, self.diarization, self.mono, 0))

    def test_resampling_and_downmix_keep_the_contract_and_the_voice(self):
        provider = self.provider()
        mono = provider.fingerprint(self.config, self.diarization, self.mono, 1)
        stereo = provider.fingerprint(self.config, self.diarization, self.stereo, 1)
        self.check_contract(stereo, stream=1)
        a, b = mono["fingerprints"][0]["embedding"], stereo["fingerprints"][0]["embedding"]
        # 44.1 kHz stereo is resampled to 16 kHz mono: the same synthetic voice, a slightly different signal.
        self.assertGreater(sum(x * y for x, y in zip(a, b)), 0.95)

    def test_a_model_directory_is_the_only_source(self):
        """The reference names a revision: a revision with no validated copy never loads anything else."""
        other = {**self.config, "fingerprint_model": {**self.config["fingerprint_model"], "revision": "f" * 40}}
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            self.provider().fingerprint(other, self.diarization, self.mono, 0)
        self.assertEqual(caught.exception.reason, "model_path_missing")

    def test_audio_the_decoder_cannot_read_is_refused_with_a_stable_cause(self):
        for name, content in [("garbage.wav", b"RIFF not really a wave file" * 20), ("empty.wav", b""), ("text.mp3", b"just some text")]:
            with self.subTest(name=name):
                path = os.path.join(self.root, name)
                Path(path).write_bytes(content)
                with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                    self.provider().fingerprint(self.config, self.diarization, path, 0)
                self.assertIn(caught.exception.reason, {"inference_failed", "model_load_failed"})
                self.assertNotIn(self.root, str(caught.exception))

    def test_a_model_with_a_non_titanet_archive_is_refused_with_its_cause(self):
        other = voice_models.model_path(self.env, REPO, "e" * 40)
        os.makedirs(other)
        write_archive(os.path.join(other, "model.nemo"), {"model_config.yaml": b"encoder: {}", "model_weights.ckpt": b"w"})
        Path(other, voice_models.READY_MARKER).write_text("e" * 40)
        config = {**self.config, "fingerprint_model": {**self.config["fingerprint_model"], "revision": "e" * 40}}
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            self.provider().fingerprint(config, self.diarization, self.mono, 0)
        self.assertEqual(caught.exception.reason, "model_load_failed")
        self.assertEqual(voice_models._cause_of_verification(caught.exception), "runtime_incompatible")
        self.assertEqual(caught.exception.detail, "variant_unsupported")

    def run_bridge(self, request):
        """The sub-process protocol of an approved runtime, exactly as voice_runtime.call starts it."""
        incoming, outgoing = os.path.join(self.root, "in.json"), os.path.join(self.root, "out.json")
        Path(incoming).write_text(json.dumps(request))
        Path(outgoing).unlink(missing_ok=True)
        worker = str(Path(voice_titanet.__file__).resolve().parent.parent)
        bootstrap = "import sys,runpy;sys.path.insert(0," + repr(worker) + ");runpy.run_module('snitch_worker.engine_bridge',run_name='__main__')"
        env = {key: value for key, value in os.environ.items() if key in {"PATH", "HOME", "TMPDIR", "LANG", "LD_LIBRARY_PATH"}}
        env.update(HF_HUB_OFFLINE="1", TRANSFORMERS_OFFLINE="1", PYTHONNOUSERSITE="1", SNITCH_VOICE_DEVICE="cpu")
        subprocess.run([sys.executable, "-I", "-c", bootstrap, incoming, outgoing], env=env, check=True, timeout=600,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        return json.loads(Path(outgoing).read_text())

    def test_the_runtime_subprocess_verifies_and_fingerprints(self):
        base = {"manifest": {"engine": "nemo-titanet", "kind": "fingerprint"}, "path": self.path,
                "manifest_path": os.path.join(self.root, voice_models.voice_manifest.MANIFEST_NAME)}
        self.assertEqual(self.run_bridge({**base, "operation": "verify"}), {"result": {}})
        answer = self.run_bridge({**base, "operation": "infer", "config": self.config, "diarization": self.diarization, "audio": self.mono, "stream": 0})
        self.check_contract(answer["result"])

    def test_the_runtime_subprocess_reports_why_a_model_is_refused(self):
        empty = os.path.join(self.root, "empty-model")
        os.makedirs(empty, exist_ok=True)
        base = {"manifest": {"engine": "nemo-titanet", "kind": "fingerprint"}, "path": empty,
                "manifest_path": os.path.join(self.root, voice_models.voice_manifest.MANIFEST_NAME), "operation": "verify"}
        self.assertEqual(self.run_bridge(base), {"error": "model_load_failed", "detail": "archive_missing"})


if __name__ == "__main__":
    unittest.main()
