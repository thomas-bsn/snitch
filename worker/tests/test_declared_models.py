import threading
import unittest
from unittest.mock import patch

from snitch_worker import declared_models, runner


class DeclaredModelsTest(unittest.TestCase):
    def declare(self, env):
        return declared_models.declared_models(env).to_dict()

    def test_nothing_configured_declares_nothing(self):
        self.assertEqual(self.declare({}), {})

    def test_transcription_reports_provider_model_and_host_port_only(self):
        declared = self.declare(
            {
                "TRANSCRIPTION_API_URL": "http://user:pass@host.docker.internal:8000/v1/x?key=abc",
                "TRANSCRIPTION_MODEL": "large-v3",
                "TRANSCRIPTION_API_KEY": "sk-should-never-be-sent-0123456789",
            }
        )
        self.assertEqual(
            declared,
            {
                "transcription": {
                    "provider": "openai-compatible",
                    "model": "large-v3",
                    "endpoint": "host.docker.internal:8000",
                }
            },
        )
        self.assertNotIn("sk-", repr(declared))
        self.assertNotIn("pass", repr(declared))

    def test_transcription_without_model_is_not_declared(self):
        self.assertEqual(
            self.declare({"TRANSCRIPTION_API_URL": "http://ollama:11434"}), {}
        )

    def test_unparseable_endpoint_is_omitted_but_model_is_kept(self):
        declared = self.declare(
            {"TRANSCRIPTION_API_URL": "http://[bad", "TRANSCRIPTION_MODEL": "m"}
        )
        self.assertEqual(
            declared, {"transcription": {"provider": "openai-compatible", "model": "m"}}
        )

    def test_voice_roles_come_from_environment(self):
        declared = self.declare(
            {
                "SNITCH_VOICE_DIARIZATION_PROVIDER": "pyannote",
                "SNITCH_VOICE_DIARIZATION_MODEL": "pyannote/speaker-diarization-community-1",
                "SNITCH_VOICE_DIARIZATION_VERSION": "4.0.1",
                "SNITCH_VOICE_FINGERPRINT_PROVIDER": "pyannote",
                "SNITCH_VOICE_FINGERPRINT_MODEL": "wespeaker-voxceleb-resnet34-LM",
            }
        )
        self.assertEqual(
            declared,
            {
                "diarization": {
                    "provider": "pyannote",
                    "model": "pyannote/speaker-diarization-community-1",
                    "version": "4.0.1",
                },
                "fingerprint": {
                    "provider": "pyannote",
                    "model": "wespeaker-voxceleb-resnet34-LM",
                },
            },
        )

    def test_values_the_server_would_refuse_are_dropped_not_sent(self):
        declared = self.declare(
            {
                "TRANSCRIPTION_MODEL": "hf_abcdefghijklmnopqrstuvwxyz",
                "SNITCH_VOICE_DIARIZATION_PROVIDER": "pyannote",
                "SNITCH_VOICE_DIARIZATION_MODEL": "m",
                "SNITCH_VOICE_DIARIZATION_VERSION": "has space",
                "SNITCH_VOICE_FINGERPRINT_PROVIDER": "x" * 200,
                "SNITCH_VOICE_FINGERPRINT_MODEL": "m",
            }
        )
        self.assertEqual(
            declared, {"diarization": {"provider": "pyannote", "model": "m"}}
        )

    def test_pulse_carries_the_declaration(self):
        env = {"TRANSCRIPTION_MODEL": "large-v3"}
        with (
            patch.dict("os.environ", env, clear=True),
            patch.object(runner, "pulse", return_value=object()) as pulse,
        ):
            stop = threading.Event()
            stop.set()
            runner.run_once(object(), 1, 45.0, stop)
        body = pulse.call_args.kwargs["body"].to_dict()
        self.assertEqual(
            body["models"],
            {"transcription": {"provider": "openai-compatible", "model": "large-v3"}},
        )


if __name__ == "__main__":
    unittest.main()
