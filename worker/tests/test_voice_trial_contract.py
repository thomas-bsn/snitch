import json
import tempfile
import unittest

from snitch_worker import voice
from snitch_client.models.worker_job_input import WorkerJobInput
from snitch_worker.runner import trial_voice_input
from tests import test_voice_nemo as nemo_tests
from tests.test_voice_real import Annotation, DiarizeOutput, model_dirs, provider

JOB = {
    "id": 7, "clip_id": 1, "step": "voicetrial.diarization.7", "input_version": "trial-7", "status": "leased",
    "priority": 0, "attempts": 1, "max_attempts": 3, "next_attempt_at": "2026-11-02T10:00:00Z", "last_error": "",
    "created_at": "2026-11-02T10:00:00Z", "updated_at": "2026-11-02T10:00:00Z", "cycle": 0, "lease_owner": None, "lease_expires_at": None, "result": None,
}


def wire(engine, repo, revision):
    """The body the server sends for a trial: exactly what jobs.trialInput encodes, with no pipeline_version."""
    return json.loads(json.dumps({
        "job": JOB, "original_name": "a.mp4", "size_bytes": 1, "video_codec": "h264", "audio_codec": "aac", "tracks": [],
        "media_url": "/m", "vocabulary": [],
        "voice_trial": {
            "config": {"provider": engine, "model": repo, "model_version": revision, "num_speakers": 2},
            "trial": {"audio_stream_index": 3},
        },
    }))


class TrialContractTest(unittest.TestCase):
    def test_pyannote_trial_decodes_and_runs(self):
        calls = []

        def pipeline(audio_path, **options):
            calls.append(options)
            return DiarizeOutput(Annotation([(0.0, 2.0, "A")]))

        body = wire("pyannote", "pyannote/speaker-diarization-community-1", "community-1")
        with model_dirs() as env:
            providers = {"pyannote": provider(env, load_pipeline=lambda path, device: pipeline)}
            decoded = trial_voice_input(WorkerJobInput.from_dict(body))
            result = voice.process_trial_step("voicetrial.diarization.7", decoded, "/tmp/a.wav", providers)
        self.assertEqual(calls, [{"num_speakers": 2}])
        self.assertEqual(result["audio_stream_index"], 3)

    def test_nemo_trial_decodes_and_runs(self):
        from snitch_worker import voice_nemo

        body = wire("nemo", nemo_tests.REPO, nemo_tests.REVISION)
        with tempfile.TemporaryDirectory() as root:
            env = nemo_tests.nemo_env(root)
            nemo_tests.active_model(env)
            nemo = voice_nemo.NemoProvider(environ=env, load_pipeline=lambda path, device: (lambda audio: ["0.0 1.0 s0"]), device_probe=lambda: "cpu")
            decoded = trial_voice_input(WorkerJobInput.from_dict(body))
            result = voice.process_trial_step("voicetrial.diarization.7", decoded, "/tmp/a.wav", {"nemo": nemo})
        self.assertEqual(result["audio_stream_index"], 3)
        self.assertEqual(len(result["speakers"]), 1)


if __name__ == "__main__":
    unittest.main()
