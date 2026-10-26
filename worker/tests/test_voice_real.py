import math
import os
import subprocess
import sys
import tempfile
import unittest
from contextlib import contextmanager
from types import SimpleNamespace
from unittest.mock import patch

from snitch_worker import runner, voice, voice_pyannote

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

        return embed

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
            raise OSError("401 token hf_SECRET at /private/models/x")

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
                raise RuntimeError("CUDA out of memory /secret")

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
