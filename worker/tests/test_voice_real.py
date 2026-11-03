import io
import json
import math
import os
import subprocess
import sys
import tempfile
import unittest
from contextlib import contextmanager
from types import SimpleNamespace
from unittest.mock import patch

from snitch_worker import log, runner, voice, voice_pyannote

CONFIG = {
    "provider": "pyannote",
    "model": "pyannote/speaker-diarization-community-1",
    "model_version": "community-1",
    "min_fingerprint_seconds": 1.0,
    "min_quality": 0.4,
}


class SimpleSegment:
    def __init__(self, start, end):
        self.start, self.end = start, end


class Annotation:
    """Mimics pyannote.core.Annotation.itertracks(yield_label=True)."""

    def __init__(self, tracks):
        self.tracks = tracks

    def itertracks(self, yield_label=False):
        for start, end, label in self.tracks:
            yield SimpleSegment(start, end), "track", label


class DiarizeOutput:
    def __init__(self, annotation):
        self.speaker_diarization = annotation


@contextmanager
def model_dirs():
    with tempfile.TemporaryDirectory() as root:
        pipeline = os.path.join(root, "diarization")
        embedding = os.path.join(root, "embedding")
        os.makedirs(pipeline)
        os.makedirs(embedding)
        yield {
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": pipeline,
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": embedding,
            "SNITCH_VOICE_EMBEDDING_MODEL_NAME": "pyannote/wespeaker-voxceleb-resnet34-LM",
            "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "wespeaker-test-1",
        }


def provider(env, **kw):
    kw.setdefault("device_probe", lambda: "cpu")
    return voice_pyannote.PyannoteProvider(environ=env, **kw)


def fake_embedder(dimensions=3):
    calls = []

    def open_audio(path):
        calls.append(path)

        def embed(start, end):
            return [start + 1.0, end - start, 2.0][:dimensions]

        # One hour of 16 kHz audio: long enough that no turn of these tests is cut.
        return voice_pyannote.DecodedAudio(embed, 16000, 1, 16000 * 3600)

    return open_audio, calls


class SelectionTests(unittest.TestCase):
    def test_configured_provider_receives_the_prepared_audio(self):
        seen = []

        class Stub:
            def ensure_ready(self, config):
                seen.append(("ready", config["provider"]))

            def diarize(self, config, audio_path, stream_index):
                seen.append(("diarize", audio_path, stream_index))
                return {"stub": True}

        result = voice.process_voice_step("voice.diarization.3", {"config": CONFIG}, 5.0, audio_path="/a.wav", providers={"pyannote": Stub()})
        self.assertEqual(result, {"stub": True})
        self.assertEqual(seen, [("ready", "pyannote"), ("diarize", "/a.wav", 3)])

    def test_fake_provider_stays_available_without_audio(self):
        config = {**CONFIG, "provider": "fake", "fake": {"speakers": 2, "dimensions": 4}}
        result = voice.process_voice_step("voice.diarization.1", {"config": config}, 4.0)
        self.assertEqual(len(result["speakers"]), 2)
        self.assertFalse(voice.requires_audio(config))
        self.assertTrue(voice.requires_audio(CONFIG))

    def test_unknown_provider_is_unavailable(self):
        with self.assertRaises(voice.VoiceProviderUnavailable):
            voice.process_voice_step("voice.diarization.1", {"config": {**CONFIG, "provider": "other"}}, 4.0, audio_path="/a.wav")

    def test_default_pyannote_without_model_paths_is_unavailable_and_redacted(self):
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            voice.process_voice_step("voice.diarization.1", {"config": CONFIG}, 4.0, audio_path="/a.wav", providers={"pyannote": provider({})})
        self.assertEqual(str(caught.exception), "voice provider is not available")
        self.assertEqual(caught.exception.reason, "model_path_missing")


class DiarizationMappingTests(unittest.TestCase):
    def diarize(self, output):
        with model_dirs() as env:
            p = provider(env, load_pipeline=lambda path, device: (lambda audio: output))
            return p.diarize(CONFIG, "/a.wav", 2)

    def test_annotation_maps_to_the_result_format_in_order_of_appearance(self):
        annotation = Annotation([(2.0, 3.0, "B"), (0.0, 1.5, "A"), (4.0, 4.5, "B"), (5.0, 5.0, "A")])
        result = self.diarize(annotation)
        self.assertEqual(result["provider"], "pyannote")
        self.assertEqual(result["model"], CONFIG["model"])
        self.assertEqual(result["model_version"], "community-1")
        self.assertEqual(result["audio_stream_index"], 2)
        self.assertEqual(
            result["speakers"],
            [
                {"label": "SPEAKER_00", "turns": [{"start": 0.0, "end": 1.5}]},
                {"label": "SPEAKER_01", "turns": [{"start": 2.0, "end": 3.0}, {"start": 4.0, "end": 4.5}]},
            ],
        )

    def test_diarize_output_object_is_supported(self):
        result = self.diarize(DiarizeOutput(Annotation([(0.0, 1.0, "x")])))
        self.assertEqual(len(result["speakers"]), 1)

    def test_empty_diarization_is_a_success(self):
        self.assertEqual(self.diarize(Annotation([]))["speakers"], [])

    def test_too_many_speakers_fail_instead_of_being_rejected_by_the_api(self):
        annotation = Annotation([(i, i + 1.0, f"s{i}") for i in range(33)])
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            self.diarize(annotation)
        self.assertEqual(caught.exception.reason, "result_too_large")


class FingerprintTests(unittest.TestCase):
    DIARIZATION = {
        "job_id": 7,
        "speakers": [
            {"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 3.0}, {"index": 1, "start": 5.0, "end": 5.3}, {"index": 2, "start": 6.0, "end": 6.8}]},
            {"label": "SPEAKER_01", "turns": [{"index": 0, "start": 3.0, "end": 4.0}]},
            {"label": "SPEAKER_02", "turns": [{"index": 0, "start": 8.0, "end": 8.2}]},
        ],
    }

    def fingerprint(self, diarization=None):
        open_audio, calls = fake_embedder()
        with model_dirs() as env:
            p = provider(env, load_embedder=lambda path, device: open_audio)
            result = p.fingerprint(CONFIG, diarization or self.DIARIZATION, "/a.wav", 2)
        return result, calls

    def test_result_has_the_fake_provider_shape(self):
        result, calls = self.fingerprint()
        self.assertEqual(result["embedding_model"], "pyannote/wespeaker-voxceleb-resnet34-LM")
        self.assertEqual(result["embedding_model_version"], "wespeaker-test-1")
        self.assertEqual(result["provider"], "pyannote")
        self.assertEqual(result["audio_stream_index"], 2)
        self.assertEqual(result["diarization_job_id"], 7)
        self.assertEqual(result["dimensions"], 3)
        self.assertEqual(calls, ["/a.wav"])
        for item in result["fingerprints"]:
            self.assertEqual(len(item["embedding"]), 3)
            self.assertAlmostEqual(math.sqrt(sum(v * v for v in item["embedding"])), 1.0, places=4)
            self.assertTrue(0 <= item["quality"] <= 1)

    def test_one_vector_per_speaker_and_per_long_enough_turn_with_duration(self):
        result, _ = self.fingerprint()
        keys = [(f["speaker_label"], f["scope"], f.get("turn_index")) for f in result["fingerprints"]]
        # 0.3 s and 0.2 s turns are below the 0.5 s embedding floor; SPEAKER_02 has nothing to embed.
        self.assertEqual(
            keys,
            [
                ("SPEAKER_00", "speaker", None),
                ("SPEAKER_00", "turn", 0),
                ("SPEAKER_00", "turn", 2),
                ("SPEAKER_01", "speaker", None),
                ("SPEAKER_01", "turn", 0),
            ],
        )
        first = result["fingerprints"][0]
        self.assertEqual((first["start"], first["end"]), (0.0, 3.8))
        turn = result["fingerprints"][1]
        self.assertEqual((turn["start"], turn["end"]), (0.0, 3.0))

    def test_quality_grows_with_duration_and_is_bounded(self):
        result, _ = self.fingerprint()
        by_key = {(f["speaker_label"], f["scope"], f.get("turn_index")): f["quality"] for f in result["fingerprints"]}
        self.assertEqual(by_key[("SPEAKER_00", "turn", 0)], 1.0)
        self.assertLess(by_key[("SPEAKER_00", "turn", 2)], by_key[("SPEAKER_01", "turn", 0)])
        self.assertLess(by_key[("SPEAKER_01", "turn", 0)], 1.0)

    def test_fingerprint_count_never_exceeds_the_api_limit(self):
        turns = [{"index": i, "start": i * 2.0, "end": i * 2.0 + 0.5 + (i % 7) * 0.1} for i in range(450)]
        result, _ = self.fingerprint({"job_id": 1, "speakers": [{"label": "SPEAKER_00", "turns": turns}]})
        self.assertLessEqual(len(result["fingerprints"]), 400)
        self.assertEqual(result["fingerprints"][0]["scope"], "speaker")

    def test_no_speaker_means_no_fingerprint_and_no_model_load(self):
        loaded = []
        with model_dirs() as env:
            p = provider(env, load_embedder=lambda path, device: loaded.append(1))
            result = p.fingerprint(CONFIG, {"job_id": 1, "speakers": []}, "/a.wav", 0)
        self.assertEqual(result["fingerprints"], [])
        self.assertEqual(loaded, [])


class AudioWindowTests(unittest.TestCase):
    """The window embedded for a turn stops at the last valid sample (clip #106: 15.19 s for 15.172646 s)."""

    RATE, SAMPLES = 16000, 160000  # 10 s
    LAST = (SAMPLES - 1) / RATE  # 9.99993750 s: the last sample pyannote accepts

    def window(self, start, end):
        return voice_pyannote.audio_window(start, end, self.SAMPLES, self.RATE)

    def test_a_segment_inside_the_audio_is_untouched(self):
        self.assertEqual(self.window(1.0, 4.0), (1.0, 4.0))

    def test_a_turn_ending_exactly_at_the_end_of_the_audio_stops_at_the_last_sample(self):
        start, end = self.window(7.0, 10.0)
        self.assertEqual((start, end), (7.0, self.LAST))
        self.assertEqual(round(end * self.RATE), self.SAMPLES - 1)

    def test_an_overshoot_is_cut_at_the_last_sample(self):
        self.assertEqual(self.window(7.0, 10.02), (7.0, self.LAST))

    def test_a_negative_start_is_clamped_to_zero(self):
        self.assertEqual(self.window(-0.02, 2.0), (0.0, 2.0))

    def test_a_segment_that_starts_after_the_audio_is_dropped(self):
        self.assertIsNone(self.window(10.5, 11.0))

    def test_a_segment_too_short_once_cut_is_dropped(self):
        # 0.7 s turn embeddable as-is, but only 0.2 s of audio left under it.
        self.assertIsNone(self.window(9.8, 10.5))

    def test_a_segment_exactly_at_the_floor_is_kept(self):
        self.assertEqual(self.window(9.0, 9.5), (9.0, 9.5))

    def test_the_window_is_computed_from_the_sample_rate_not_from_seconds(self):
        # 44.1 kHz: the last sample is at (N - 1) / 44100, not at a rounded duration.
        self.assertEqual(voice_pyannote.audio_window(0.0, 10.0, 441000, 44100), (0.0, 440999 / 44100))


class FakeTorchAudioModules:
    """Stands in for torch, torchaudio and pyannote so the real ``_load_embedder`` closure runs without models.

    ``Inference.crop`` rejects segments that end after the audio, as pyannote does.
    """

    def __init__(self, duration):
        self.duration = duration
        self.crops = []

    def modules(self):
        outer = self

        class Waveform:
            shape = (1, int(round(outer.duration * 1_000_000)))  # 1 MHz: the duration survives exactly

        class Crop:
            def __init__(self, values):
                self.values = values

            def reshape(self, _):
                return self

            def tolist(self):
                return self.values

        class Inference:
            def __init__(self, model, window, device):
                pass

            def crop(self, audio, segment):
                outer.crops.append((segment.start, segment.end))
                if round(segment.end * 1_000_000) >= round(outer.duration * 1_000_000):  # strictly before the last sample
                    raise ValueError("requested chunk with end time greater than file duration")
                return Crop([segment.start + 1.0, segment.end - segment.start, 2.0])

        class Segment:
            def __init__(self, start, end):
                self.start, self.end = start, end

        torch = SimpleNamespace(device=lambda name: name)
        torchaudio = SimpleNamespace(load=lambda path: (Waveform(), 1_000_000))
        pyannote_audio = SimpleNamespace(Inference=Inference, Model=SimpleNamespace(from_pretrained=lambda path: None))
        pyannote_core = SimpleNamespace(Segment=Segment)
        pyannote = SimpleNamespace(audio=pyannote_audio, core=pyannote_core)
        return {
            "torch": torch,
            "torchaudio": torchaudio,
            "pyannote": pyannote,
            "pyannote.audio": pyannote_audio,
            "pyannote.core": pyannote_core,
        }


class OvershootFingerprintTests(unittest.TestCase):
    CLIP = 15.172646

    def fingerprint(self, turns):
        fake = FakeTorchAudioModules(self.CLIP)
        diarization = {"job_id": 9, "speakers": [{"label": "SPEAKER_00", "turns": turns}]}
        with model_dirs() as env, patch.dict(sys.modules, fake.modules()):
            p = provider(env, load_embedder=voice_pyannote._load_embedder)
            result = p.fingerprint(CONFIG, diarization, "/clip.wav", 0)
        return result, fake

    def test_a_turn_ending_after_the_wav_is_embedded_within_the_audio(self):
        result, fake = self.fingerprint([{"index": 0, "start": 12.0, "end": 15.19}])
        self.assertEqual(len(fake.crops), 1)  # one crop per turn, bounded to the audio
        self.assertAlmostEqual(fake.crops[0][1], self.CLIP - 1e-6, places=9)  # last sample, not the rounded duration
        turn = [f for f in result["fingerprints"] if f["scope"] == "turn"][0]
        self.assertEqual((turn["start"], turn["end"]), (12.0, 15.19))  # original timestamps kept

    def test_a_turn_ending_exactly_on_the_last_sample_is_embedded(self):
        # The case PR #231 did not cover: the bound was the duration itself, which pyannote refuses.
        result, fake = self.fingerprint([{"index": 0, "start": 12.0, "end": self.CLIP}])
        self.assertEqual(len(fake.crops), 1)
        self.assertTrue(any(f["scope"] == "turn" for f in result["fingerprints"]))

    def test_a_turn_starting_after_the_wav_is_skipped_without_failing_the_clip(self):
        result, fake = self.fingerprint([{"index": 0, "start": 0.0, "end": 4.0}, {"index": 1, "start": 15.4, "end": 16.2}])
        self.assertEqual([f["scope"] for f in result["fingerprints"]], ["speaker", "turn"])
        self.assertEqual(result["fingerprints"][1]["turn_index"], 0)
        self.assertTrue(all(end <= self.CLIP + 1e-9 for _, end in fake.crops))

    def test_a_speaker_whose_only_turns_fall_outside_the_audio_gets_no_fingerprint(self):
        result, _ = self.fingerprint([{"index": 0, "start": 15.3, "end": 15.9}])
        self.assertEqual(result["fingerprints"], [])


class StrictAudio:
    """Mimics pyannote's in-memory crop: a chunk must end strictly before the last sample."""

    def __init__(self, seconds, sample_rate=16000, channels=1):
        self.sample_rate, self.channels = sample_rate, channels
        self.num_samples = round(seconds * sample_rate)
        self.calls = []

    def __call__(self, start, end):
        if round(end * self.sample_rate) >= self.num_samples:
            raise ValueError(f"requested chunk with end time (t={end:.3f}s) greater than /clips/secret.wav file duration")
        self.calls.append((start, end))
        return [1.0, end - start, 2.0]


class AudioBoundsTests(unittest.TestCase):
    """A diarization turn can end on, or just past, the last sample of the clip (speech up to the end)."""

    def fingerprint(self, audio, turns):
        diarization = {"job_id": 40, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": i, "start": s, "end": e} for i, (s, e) in enumerate(turns)]}]}
        with model_dirs() as env:
            p = provider(env, load_embedder=lambda path, device: lambda audio_path: audio)
            return p.fingerprint(CONFIG, diarization, "/a.wav", 1)

    def test_a_turn_ending_at_the_end_of_the_clip_is_embedded(self):
        audio = StrictAudio(10.0)
        result = self.fingerprint(audio, [(2.0, 4.0), (7.0, 10.0)])
        self.assertEqual([f.get("turn_index") for f in result["fingerprints"]], [None, 0, 1])
        self.assertLess(audio.calls[-1][1], 10.0)
        self.assertGreater(audio.calls[-1][1], 9.99)

    def test_a_turn_ending_past_the_end_of_the_clip_is_cut_at_the_end(self):
        audio = StrictAudio(10.0)
        result = self.fingerprint(audio, [(7.0, 10.02)])
        self.assertEqual([f.get("turn_index") for f in result["fingerprints"]], [None, 0])
        # The stored turn keeps its diarization bounds; only the embedded window is cut.
        self.assertEqual(result["fingerprints"][1]["end"], 10.02)

    def test_a_turn_too_short_once_cut_is_skipped_rather_than_failing(self):
        audio = StrictAudio(10.0)
        result = self.fingerprint(audio, [(2.0, 4.0), (9.8, 10.5)])
        self.assertEqual([f.get("turn_index") for f in result["fingerprints"]], [None, 0])
        self.assertEqual(audio.calls, [(2.0, 4.0)])


class FingerprintDiagnosticLogTests(unittest.TestCase):
    """The step's failure reaches the server as a bare code: the Worker log must say why, without secrets."""

    DIARIZATION = {"job_id": 40, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 1.0, "end": 3.0}, {"index": 1, "start": 4.0, "end": 4.2}]}]}

    def tearDown(self):
        log.configure("INFO", stream=io.StringIO())

    def failing_fingerprint(self, load_embedder):
        stream = io.StringIO()
        log.configure("INFO", stream=stream)
        with model_dirs() as env:
            env["HF_TOKEN"] = "hf_TOPSECRETvalue123"
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                provider(env, load_embedder=load_embedder).fingerprint(CONFIG, self.DIARIZATION, "/clips/private/clip-40.wav", 1)
        lines = [json.loads(line) for line in stream.getvalue().splitlines()]
        return caught.exception, [line for line in lines if line["event"] == "voice_fingerprint_failed"], stream.getvalue()

    def test_an_inference_error_is_logged_with_its_stage_class_and_audio_context(self):
        def explode(start, end):
            raise RuntimeError("shape mismatch for /models/wespeaker/pytorch_model.bin token=hf_TOPSECRETvalue123 https://huggingface.co/x")

        audio = voice_pyannote.DecodedAudio(explode, 48000, 2, 48000 * 10)
        exc, [line], raw = self.failing_fingerprint(lambda path, device: lambda audio_path: audio)
        self.assertEqual(exc.reason, "inference_failed")
        self.assertEqual(line["level"], "WARN")
        self.assertEqual(line["error_code"], "inference_failed")
        self.assertEqual(line["exc_class"], "RuntimeError")
        self.assertIn("shape mismatch", line["error_summary"])
        self.assertEqual(line["stage"], "embedding_inference")
        self.assertEqual((line["sample_rate"], line["channels"], line["audio_duration_s"]), (48000, 2, 10.0))
        self.assertEqual((line["segments"], line["embeddable_segments"]), (2, 1))
        self.assertEqual((line["shortest_segment_s"], line["longest_segment_s"]), (0.2, 2.0))
        self.assertEqual((line["segment_start"], line["segment_end"]), (1.0, 3.0))
        self.assertEqual(line["input_shape"], [2, 96000])
        self.assertEqual(line["device"], "cpu")
        for secret in ("hf_TOPSECRET", "/models/", "huggingface.co", "clip-40", "/clips/"):
            self.assertNotIn(secret, raw)

    def test_an_audio_decoding_error_is_logged_as_such(self):
        def open_audio(audio_path):
            raise RuntimeError(f"could not decode {audio_path}")

        exc, [line], raw = self.failing_fingerprint(lambda path, device: open_audio)
        self.assertEqual(exc.reason, "inference_failed")
        self.assertEqual(line["stage"], "audio_decode")
        self.assertNotIn("clip-40", raw)

    def test_a_model_load_error_keeps_its_code_and_logs_the_original_class(self):
        def load_embedder(path, device):
            raise OSError(f"cannot read {path} with Authorization: Bearer abc.def")

        exc, [line], raw = self.failing_fingerprint(load_embedder)
        self.assertEqual(exc.reason, "model_load_failed")
        self.assertEqual((line["stage"], line["exc_class"], line["error_code"]), ("model_load", "OSError", "model_load_failed"))
        self.assertNotIn("abc.def", raw)
        self.assertIsNone(exc.__cause__)


class InferenceDiagnosticLogTests(unittest.TestCase):
    def test_an_inference_failure_logs_its_code_and_class_but_never_its_text(self):
        class CudaBoom(Exception):
            pass

        def pipeline(path, **options):
            raise CudaBoom("secret path /private/x and token hf_SECRET")

        with model_dirs() as env, self.assertLogs("snitch_worker", level="WARNING") as logs:
            p = provider(env, load_pipeline=lambda path, device: pipeline)
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                p.diarize(CONFIG, "/a.wav", 0)
        self.assertEqual(caught.exception.reason, "inference_failed")
        event = logs.records[0]
        self.assertEqual(event.getMessage(), "diarization_inference_failed")
        self.assertEqual(event.fields["code"], "inference_failed")
        self.assertEqual(event.fields["exc_class"], "CudaBoom")
        self.assertNotIn("hf_SECRET", str(event.fields))
        self.assertNotIn("/private", str(event.fields))

    def test_a_fingerprint_failure_is_logged_with_the_same_safe_fields(self):
        def broken_open(path):
            def embed(start, end):
                raise RuntimeError("cuda out of memory on /private/x")

            return voice_pyannote.DecodedAudio(embed, 16000, 1, 16000 * 60)

        with model_dirs() as env, self.assertLogs("snitch_worker", level="WARNING") as logs:
            p = provider(env, load_embedder=lambda path, device: broken_open)
            diarization = {"job_id": 1, "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 2.0}]}]}
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                p.fingerprint(CONFIG, diarization, "/a.wav", 0)
        self.assertEqual(caught.exception.reason, "gpu_out_of_memory")
        self.assertEqual(logs.records[0].fields["exc_class"], "RuntimeError")
        self.assertNotIn("/private", str(logs.records[0].fields))


class RobustnessTests(unittest.TestCase):
    def test_models_are_loaded_once_per_process(self):
        loads = []

        def load_pipeline(path, device):
            loads.append(device)
            return lambda audio: Annotation([(0.0, 1.0, "a")])

        with model_dirs() as env:
            p = provider(env, load_pipeline=load_pipeline)
            p.diarize(CONFIG, "/a.wav", 0)
            p.diarize(CONFIG, "/b.wav", 0)
        self.assertEqual(loads, ["cpu"])

    def test_loader_errors_are_redacted(self):
        def broken(path, device):
            raise OSError("disk token hf_SECRET at /private/models/x")

        with model_dirs() as env:
            p = provider(env, load_pipeline=broken)
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                p.diarize(CONFIG, "/a.wav", 0)
        self.assertNotIn("hf_SECRET", repr(caught.exception.args))
        self.assertIsNone(caught.exception.__cause__)
        self.assertTrue(caught.exception.__suppress_context__)
        self.assertEqual(caught.exception.reason, "model_load_failed")

    def test_inference_errors_are_redacted(self):
        def load_pipeline(path, device):
            def run(audio):
                raise RuntimeError("tensor shape mismatch /secret")

            return run

        with model_dirs() as env:
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                provider(env, load_pipeline=load_pipeline).diarize(CONFIG, "/a.wav", 0)
        self.assertEqual(caught.exception.reason, "inference_failed")

    def test_missing_model_directory_is_reported_without_the_path(self):
        env = {"SNITCH_VOICE_DIARIZATION_MODEL_PATH": "/does/not/exist", "SNITCH_VOICE_EMBEDDING_MODEL_PATH": "/does/not/exist"}
        with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
            provider(env).ensure_ready(CONFIG)
        self.assertEqual(caught.exception.reason, "model_path_missing")
        self.assertNotIn("/does", str(caught.exception))

    def test_embedding_version_is_required(self):
        with model_dirs() as env:
            del env["SNITCH_VOICE_EMBEDDING_MODEL_VERSION"]
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                provider(env).ensure_ready(CONFIG)
        self.assertEqual(caught.exception.reason, "embedding_version_missing")

    def test_explicit_cuda_without_gpu_fails_clearly_and_auto_uses_the_gpu(self):
        with model_dirs() as env:
            with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                provider({**env, "SNITCH_VOICE_DEVICE": "cuda"}, load_pipeline=lambda p, d: None).diarize(CONFIG, "/a.wav", 0)
            self.assertEqual(caught.exception.reason, "cuda_unavailable")
            devices = []

            def load_pipeline(path, device):
                devices.append(device)
                return lambda audio: Annotation([])

            provider(env, device_probe=lambda: "cuda", load_pipeline=load_pipeline).diarize(CONFIG, "/a.wav", 0)
            self.assertEqual(devices, ["cuda"])

    def test_loading_forces_offline_mode_and_never_reads_the_token(self):
        with model_dirs() as env:
            env["HF_TOKEN"] = "hf_SECRET"
            env["HF_HUB_OFFLINE"] = "0"
            provider(env, load_pipeline=lambda p, d: (lambda a: Annotation([]))).diarize(CONFIG, "/a.wav", 0)
            self.assertEqual(env["HF_HUB_OFFLINE"], "1")
            self.assertEqual(env["TRANSFORMERS_OFFLINE"], "1")

    def test_importing_the_worker_voice_modules_does_not_import_ml_libraries(self):
        code = "import sys, snitch_worker.voice, snitch_worker.voice_pyannote; bad=[m for m in ('torch','torchaudio','pyannote','speechbrain') if m in sys.modules]; sys.exit(1 if bad else 0)"
        run = subprocess.run([sys.executable, "-c", code], env={**os.environ, "PYTHONPATH": os.pathsep.join(sys.path)}, capture_output=True)
        self.assertEqual(run.returncode, 0, run.stderr)


class RunnerAudioTests(unittest.TestCase):
    def run_step(self, step, config, providers, voice_extra=None):
        streamed = []

        class Media:
            def __enter__(self):
                return self

            def __exit__(self, *a):
                return False

            def raise_for_status(self):
                pass

            def iter_bytes(self):
                return [b"media"]

        def stream(method, url, headers=None):
            streamed.append((method, url, headers))
            return Media()

        http_client = SimpleNamespace(stream=stream)
        client = SimpleNamespace(_base_url="https://snitch.example", get_httpx_client=lambda: http_client)
        task = SimpleNamespace(id=9, step=step, input_version="d5-a2-pvoice-x")
        claim = SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")
        voice_input = {"config": config, **(voice_extra or {})}
        task_input = SimpleNamespace(voice=SimpleNamespace(to_dict=lambda: voice_input), duration_seconds=4.0, media_url="/api/worker/jobs/9/media")
        prepared = []
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=claim),
            patch.object(runner, "heartbeat_job", return_value=object()),
            patch.object(runner, "get_job_input", return_value=task_input),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "prepare_audio", side_effect=lambda media, wav, index: prepared.append((media, wav, index))),
        ):
            self.assertTrue(runner.run_once(client, capacity=1, voice_providers=providers))
        return complete, fail, streamed, prepared

    def test_real_provider_gets_the_prepared_audio_of_the_leased_media(self):
        seen = []

        class Stub:
            def ensure_ready(self, config):
                pass

            def diarize(self, config, audio_path, stream_index):
                seen.append((audio_path, stream_index))
                return {"provider": "pyannote", "speakers": []}

        complete, fail, streamed, prepared = self.run_step("voice.diarization.2", CONFIG, {"pyannote": Stub()})
        fail.assert_not_called()
        self.assertEqual(streamed[0][1], "https://snitch.example/api/worker/jobs/9/media")
        self.assertEqual(streamed[0][2], {"X-Worker-Lease-Token": "lt_synthetic-lease-token"})
        self.assertEqual(prepared[0][2], 2)
        self.assertEqual(seen, [(prepared[0][1], 2)])
        self.assertEqual(complete.call_args.kwargs["body"].result.to_dict()["provider"], "pyannote")

    def test_unready_provider_fails_before_any_media_is_downloaded(self):
        class Unready:
            def ensure_ready(self, config):
                raise voice.VoiceProviderUnavailable("model_path_missing")

        complete, fail, streamed, prepared = self.run_step("voice.fingerprint.2", CONFIG, {"pyannote": Unready()}, {"diarization": {"job_id": 1, "speakers": []}})
        complete.assert_not_called()
        self.assertEqual(streamed, [])
        self.assertEqual(prepared, [])
        self.assertEqual(fail.call_args.kwargs["body"].error, "provider_unavailable")


class DiarizationDiagnosticTests(RunnerAudioTests):
    """A diarization failure has its own catalogue code and typed diagnostic; a run logs what ran, never content."""

    PRIVATE = "synthetic words token=hf_secret /tmp/private/audio.wav"

    def capture_logs(self):
        import io
        import json as jsonlib

        from snitch_worker import log

        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        return lambda: [jsonlib.loads(line) for line in stream.getvalue().splitlines()]

    def test_each_diarization_failure_has_its_own_code_and_diagnostic(self):
        for reason, expected in (
            ("model_path_missing", "pyannote_model_missing"),
            ("model_access_denied", "pyannote_access_denied"),
            ("cuda_unavailable", "gpu_unavailable"),
            ("gpu_out_of_memory", "gpu_out_of_memory"),
            ("inference_failed", "diarization_inference_failed"),
        ):
            with self.subTest(reason):
                logs = self.capture_logs()

                class Failing:
                    def ensure_ready(self, config):
                        pass

                    def diarize(self, config, audio_path, stream_index):
                        raise voice.VoiceProviderUnavailable(reason)

                _complete, fail, _s, _p = self.run_step("voice.diarization.0", CONFIG, {"pyannote": Failing()})
                payload = fail.call_args.kwargs["body"].to_dict()
                self.assertEqual(payload["error"], expected)
                self.assertEqual((payload["provider"], payload["model"], payload["revision"]), ("pyannote", CONFIG["model"], CONFIG["model_version"]))
                self.assertGreaterEqual(payload["duration_ms"], 0)
                failed = [line for line in logs() if line["event"] == "job_failed"]
                self.assertEqual(failed[0]["error_code"], expected)

    def test_a_failed_diarization_leaves_a_trace_without_content_token_signed_url_or_vector(self):
        logs = self.capture_logs()
        stream_text = []
        secrets = ("lt_synthetic-lease-token", "hf_secret", "X-Amz-Signature=abc123", "bonjour tout le monde", "0.31415927")

        class Leaky:
            def ensure_ready(self, config):
                pass

            def diarize(self, config, audio_path, stream_index):
                raise RuntimeError("CUDA out of memory for https://storage.example/a.wav?X-Amz-Signature=abc123 token=hf_secret "
                                   "transcript=bonjour tout le monde embedding=[0.31415927]")

        _complete, fail, _s, _p = self.run_step("voice.diarization.0", CONFIG, {"pyannote": Leaky()})
        stream_text.append(json.dumps(logs()))
        stream_text.append(json.dumps(fail.call_args.kwargs["body"].to_dict()))
        failed = [line for line in logs() if line["event"] == "job_failed"]
        self.assertEqual(len(failed), 1)
        self.assertEqual((failed[0]["job_id"], failed[0]["step"]), (9, "voice.diarization.0"))
        for text in stream_text:
            for secret in secrets:
                self.assertNotIn(secret, text)

    def test_a_preflight_failure_is_categorised_too(self):
        class Unready:
            def ensure_ready(self, config):
                raise voice.VoiceProviderUnavailable("model_path_missing")

        _c, fail, _s, _p = self.run_step("voice.diarization.0", CONFIG, {"pyannote": Unready()})
        self.assertEqual(fail.call_args.kwargs["body"].error, "pyannote_model_missing")

    def test_a_successful_diarization_logs_provider_model_speakers_and_segments_only(self):
        logs = self.capture_logs()

        class Ok:
            def ensure_ready(self, config):
                pass

            def diarize(self, config, audio_path, stream_index):
                return {"provider": "pyannote", "speakers": [
                    {"label": "SPEAKER_00", "turns": [{"start": 0, "end": 1}, {"start": 2, "end": 3}]},
                    {"label": "SPEAKER_01", "turns": [{"start": 1, "end": 2}]},
                ]}

        _c, fail, _s, _p = self.run_step("voice.diarization.0", CONFIG, {"pyannote": Ok()})
        fail.assert_not_called()
        done = [line for line in logs() if line["event"] == "diarization_done"]
        self.assertEqual(len(done), 1)
        self.assertEqual((done[0]["provider"], done[0]["model"], done[0]["revision"]), ("pyannote", CONFIG["model"], CONFIG["model_version"]))
        self.assertEqual((done[0]["speakers"], done[0]["segments"]), (2, 3))
        self.assertIn("job_id", done[0])
        self.assertGreaterEqual(done[0]["duration_ms"], 0)

    def test_provider_logs_device_load_and_inference_times(self):
        logs = self.capture_logs()
        with model_dirs() as env:
            provider(env, load_pipeline=lambda path, device: (lambda audio: Annotation([]))).diarize(CONFIG, "/a.wav", 0)
        events = {line["event"]: line for line in logs()}
        self.assertEqual(events["diarization_model_loaded"]["device"], "cpu")
        self.assertGreaterEqual(events["diarization_model_loaded"]["load_ms"], 0)
        self.assertGreaterEqual(events["diarization_inference_done"]["inference_ms"], 0)
        self.assertEqual(events["diarization_inference_done"]["device"], "cpu")

    def test_provider_classifies_memory_and_access_errors_without_their_text(self):
        class GatedRepoError(Exception):
            pass

        def oom_load(path, device):
            raise RuntimeError("CUDA out of memory " + self.PRIVATE)

        def denied_load(path, device):
            raise GatedRepoError("403 " + self.PRIVATE)

        def oom_run(path, device):
            def run(audio):
                raise RuntimeError("CUDA out of memory " + self.PRIVATE)

            return run

        for name, loader, reason in (("oom load", oom_load, "gpu_out_of_memory"), ("denied", denied_load, "model_access_denied"), ("oom inference", oom_run, "gpu_out_of_memory")):
            with self.subTest(name), model_dirs() as env:
                with self.assertRaises(voice.VoiceProviderUnavailable) as caught:
                    provider(env, load_pipeline=loader).diarize(CONFIG, "/a.wav", 0)
                self.assertEqual(caught.exception.reason, reason)

    def test_classify_prefers_type_and_status_and_ignores_identifiers_that_look_like_a_status(self):
        class HubError(Exception):
            def __init__(self, status):
                super().__init__("opaque " + self.__class__.__name__)
                self.response = SimpleNamespace(status_code=status)

        cases = (
            ("status 401 on response", HubError(401), "model_access_denied"),
            ("status 403 on response", HubError(403), "model_access_denied"),
            ("status 500 with 401 in text", _with_text(HubError(500), "HTTP 401 echoed"), "inference_failed"),
            ("memory error type", MemoryError(), "gpu_out_of_memory"),
            ("identifier holding 401", RuntimeError("clip 401 of batch req-401-x failed"), "inference_failed"),
            ("identifier glued to 403", RuntimeError("trace a403b2 failed"), "inference_failed"),
            ("last resort text", RuntimeError("HTTP 401 Client Error"), "model_access_denied"),
        )
        for name, exc, expected in cases:
            with self.subTest(name):
                self.assertEqual(voice_pyannote._classify(exc, "inference_failed"), expected)


def _with_text(exc, text):
    exc.args = (text,)
    return exc


class DownloadScriptTests(unittest.TestCase):
    def load(self):
        import importlib.util

        path = os.path.join(os.path.dirname(__file__), "..", "scripts", "download_models.py")
        spec = importlib.util.spec_from_file_location("download_models", path)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module

    def test_refuses_without_explicit_confirmation_or_token_and_never_downloads(self):
        script = self.load()
        calls = []
        args = ["--diarization-dir", "/d", "--embedding-dir", "/e"]
        self.assertEqual(script.main(args, {"HF_TOKEN": "t"}, downloader=lambda **k: calls.append(k)), 2)
        self.assertEqual(script.main(args + ["--confirm-terms"], {}, downloader=lambda **k: calls.append(k)), 2)
        self.assertEqual(calls, [])

    def test_downloads_both_models_and_hides_errors(self):
        script = self.load()
        calls = []
        args = ["--diarization-dir", "/d", "--embedding-dir", "/e", "--confirm-terms"]
        self.assertEqual(script.main(args, {"HF_TOKEN": "t"}, downloader=lambda **k: calls.append(k["repo_id"])), 0)
        self.assertEqual(calls, [script.DIARIZATION_REPO, script.EMBEDDING_REPO])

        def boom(**k):
            raise OSError("401 hf_SECRET")

        self.assertEqual(script.main(args, {"HF_TOKEN": "hf_SECRET"}, downloader=boom), 1)


if __name__ == "__main__":
    unittest.main()
