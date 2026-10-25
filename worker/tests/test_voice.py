import math
import unittest
from types import SimpleNamespace
from unittest.mock import patch

from snitch_worker import runner, voice

CONFIG = {
    "provider": "fake",
    "model": "fixture-diarization",
    "model_version": "fixture-v1",
    "fake": {"speakers": 2, "dimensions": 4},
}


class FakeVoiceProviderTests(unittest.TestCase):
    def test_diarization_is_deterministic_and_covers_the_clip(self):
        first = voice.process_voice_step("voice.diarization.2", {"config": CONFIG}, 4.0)
        again = voice.process_voice_step("voice.diarization.2", {"config": CONFIG}, 4.0)
        self.assertEqual(first, again)
        self.assertEqual(first["audio_stream_index"], 2)
        self.assertEqual([s["label"] for s in first["speakers"]], ["SPEAKER_00", "SPEAKER_01"])
        self.assertEqual(first["speakers"][0]["turns"], [{"start": 0.0, "end": 1.0}, {"start": 2.0, "end": 3.0}])
        self.assertEqual(first["model"], "fixture-diarization")

    def test_fingerprints_follow_the_pinned_diarization_with_unit_vectors(self):
        diarization = {
            "job_id": 7,
            "speakers": [{"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 1.0}, {"index": 1, "start": 2.0, "end": 2.2}]}],
        }
        result = voice.process_voice_step("voice.fingerprint.2", {"config": CONFIG, "diarization": diarization}, None)
        self.assertEqual(result["diarization_job_id"], 7)
        self.assertEqual(result["dimensions"], 4)
        scopes = [(f["scope"], f.get("turn_index")) for f in result["fingerprints"]]
        self.assertEqual(scopes, [("speaker", None), ("turn", 0), ("turn", 1)])
        self.assertAlmostEqual(result["fingerprints"][0]["end"], 1.2)
        self.assertLess(result["fingerprints"][2]["quality"], 0.5)
        for fingerprint in result["fingerprints"]:
            self.assertEqual(len(fingerprint["embedding"]), 4)
            self.assertAlmostEqual(math.sqrt(sum(v * v for v in fingerprint["embedding"])), 1.0, places=4)

    def test_other_providers_never_run_or_download_anything(self):
        config = {**CONFIG, "provider": "pyannote"}
        with self.assertRaises(voice.VoiceProviderUnavailable):
            voice.process_voice_step("voice.diarization.2", {"config": config}, 4.0)
        self.assertTrue(voice.is_voice_step("voice.fingerprint.1"))
        self.assertFalse(voice.is_voice_step("transcription.audio.1"))


class VoiceRunnerTests(unittest.TestCase):
    def run_step(self, step, voice_input, duration=4.0):
        http_client = SimpleNamespace(get=lambda *_a, **_k: self.fail("media requested"))
        client = SimpleNamespace(_base_url="https://snitch.example", get_httpx_client=lambda: http_client)
        task = SimpleNamespace(id=9, step=step, input_version="d5-a2-pvoice-x")
        claim = SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")
        task_input = SimpleNamespace(voice=SimpleNamespace(to_dict=lambda: voice_input), duration_seconds=duration)
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=claim),
            patch.object(runner, "heartbeat_job", return_value=object()),
            patch.object(runner, "get_job_input", return_value=task_input),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
        ):
            self.assertTrue(runner.run_once(client, capacity=1))
        return complete, fail

    def test_runner_completes_a_fake_diarization_without_media(self):
        complete, fail = self.run_step("voice.diarization.2", {"config": CONFIG})
        fail.assert_not_called()
        result = complete.call_args.kwargs["body"].result.to_dict()
        self.assertEqual(len(result["speakers"]), 2)

    def test_runner_reports_only_a_redacted_code_for_unavailable_providers(self):
        complete, fail = self.run_step("voice.diarization.2", {"config": {**CONFIG, "provider": "pyannote"}})
        complete.assert_not_called()
        self.assertEqual(fail.call_args.kwargs["body"].error, "provider_unavailable")


if __name__ == "__main__":
    unittest.main()
