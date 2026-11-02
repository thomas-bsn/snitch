import unittest

from snitch_worker import voice
from tests.test_voice_real import Annotation, DiarizeOutput, model_dirs, provider

CONFIG = {
    "provider": "pyannote",
    "model": "pyannote/speaker-diarization-community-1",
    "model_version": "community-1",
}


class Recorder:
    """A fake pipeline that records the options it is called with."""

    def __init__(self):
        self.calls = []

    def __call__(self, audio_path, **options):
        self.calls.append((audio_path, options))
        return DiarizeOutput(Annotation([(0.0, 2.0, "A"), (2.0, 4.0, "B")]))


def trial_provider(env, pipeline):
    return provider(env, load_pipeline=lambda path, device: pipeline)


class DiarizationTrialTest(unittest.TestCase):
    def test_num_speakers_is_passed_to_the_pipeline(self):
        with model_dirs() as env:
            pipeline = Recorder()
            result = trial_provider(env, pipeline).diarize({**CONFIG, "num_speakers": 3}, "/tmp/a.wav", 2)
        self.assertEqual(pipeline.calls, [("/tmp/a.wav", {"num_speakers": 3})])
        self.assertEqual(len(result["speakers"]), 2)

    def test_min_and_max_speakers_are_passed_to_the_pipeline(self):
        with model_dirs() as env:
            pipeline = Recorder()
            trial_provider(env, pipeline).diarize({**CONFIG, "min_speakers": 2, "max_speakers": 4}, "/tmp/a.wav", 2)
        self.assertEqual(pipeline.calls[0][1], {"min_speakers": 2, "max_speakers": 4})

    def test_a_regular_job_calls_the_pipeline_without_option(self):
        with model_dirs() as env:
            pipeline = Recorder()
            trial_provider(env, pipeline).diarize(CONFIG, "/tmp/a.wav", 2)
        self.assertEqual(pipeline.calls, [("/tmp/a.wav", {})])

    def test_trial_step_diarizes_the_trial_stream_with_its_options(self):
        with model_dirs() as env:
            pipeline = Recorder()
            providers = {"pyannote": trial_provider(env, pipeline)}
            payload = {"config": {**CONFIG, "num_speakers": 2}, "trial": {"audio_stream_index": 3}}
            result = voice.process_trial_step("voicetrial.diarization.7", payload, "/tmp/a.wav", providers)
        self.assertEqual(result["audio_stream_index"], 3)
        self.assertEqual(pipeline.calls[0][1], {"num_speakers": 2})

    def test_trial_step_never_runs_the_fake_provider_or_a_regular_step(self):
        with self.assertRaises(voice.VoiceProviderUnavailable):
            voice.process_trial_step("voicetrial.diarization.7", {"config": {"provider": "fake"}}, "/tmp/a.wav")
        with self.assertRaises(ValueError):
            voice.process_trial_step("voice.diarization.2", {"config": CONFIG}, "/tmp/a.wav", {})
        self.assertTrue(voice.is_trial_step("voicetrial.diarization.7"))
        self.assertFalse(voice.is_voice_step("voicetrial.diarization.7"))


if __name__ == "__main__":
    unittest.main()
