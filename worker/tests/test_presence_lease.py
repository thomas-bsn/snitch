import contextlib
import inspect
import io
import os
import tempfile
import threading
import time
import unittest
import wave
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import httpx
from snitch_client.models.error import Error

from snitch_worker import log, runner, voice_nemo, voice_pyannote

# Set once a fake inference is running, so the server refuses the lease mid-inference.
inference_started = []


def _long_inference(*_args, **_kwargs):
    """A model that computes for a minute in Python, as torch does between its layers."""
    inference_started.append(1)
    deadline = time.monotonic() + 60
    total = 0
    while time.monotonic() < deadline:
        total += sum(range(100))
    return total


def _model_env(root):
    env = {
        "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(root, "diarization"),
        "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(root, "embedding"),
        "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "wespeaker-test-1",
    }
    os.makedirs(env["SNITCH_VOICE_DIARIZATION_MODEL_PATH"])
    os.makedirs(env["SNITCH_VOICE_EMBEDDING_MODEL_PATH"])
    with open(os.path.join(env["SNITCH_VOICE_DIARIZATION_MODEL_PATH"], "model.nemo"), "w"):
        pass
    return env


def _write_silence(path, seconds=1):
    with wave.open(path, "wb") as audio:
        audio.setnchannels(1)
        audio.setsampwidth(2)
        audio.setframerate(16000)
        audio.writeframes(b"\0\0" * 16000 * seconds)


class PresenceAndLeaseTests(unittest.TestCase):
    """ADR 0029: presence is independent of jobs; a lost lease cancels without a result."""

    LEASE = "lt_synthetic-lease-token"

    def setUp(self):
        self.addCleanup(log.configure, "INFO", io.StringIO())
        log.configure("INFO", io.StringIO())
        self.client = SimpleNamespace(_base_url="https://snitch.example")
        self.task = SimpleNamespace(
            id=7, clip_id=3, step="fixture.echo", input_version="v1", cycle=1, attempts=1, lease_owner="wk_synthetic"
        )
        self.claim = SimpleNamespace(job=self.task, lease_token=self.LEASE)

    def test_presence_retry_delay_grows_and_is_capped_at_ten_seconds(self):
        delays = [runner.presence_retry_delay(attempt) for attempt in range(1, 8)]
        self.assertEqual(delays, [1, 2, 4, 8, 10, 10, 10])

    def test_by_default_a_refused_lease_is_seen_within_ten_seconds(self):
        # ADR 0029: a refusal cancels the job "immédiatement"; the renewal is what hears it. The
        # server lease lasts 2 minutes, so a network error still leaves many retries.
        for entry in (runner.run_once, runner.run_until_stopped):
            interval = inspect.signature(entry).parameters["heartbeat_interval"].default
            self.assertLessEqual(interval, 10, entry.__name__)
            self.assertGreaterEqual(120 / interval, 10, entry.__name__)

    def test_presence_pulses_on_its_own_thread_while_a_job_runs(self):
        stop = threading.Event()
        release = threading.Event()
        pulses = []

        def long_job(*_args, **_kwargs):
            release.wait(2)
            return True

        def record_pulse(**_kwargs):
            pulses.append(time.monotonic())
            if len(pulses) >= 4:
                release.set()
                stop.set()
            return object()

        with (
            patch.object(runner, "pulse", side_effect=record_pulse),
            patch.object(runner, "run_once", side_effect=long_job),
        ):
            runner.run_until_stopped(self.client, 1, stop, poll_interval=0.01, presence_interval=0.01)
        self.assertGreaterEqual(len(pulses), 4)

    def test_presence_keeps_pulsing_after_network_errors(self):
        stop = threading.Event()
        calls = []

        def flaky_pulse(**_kwargs):
            calls.append(1)
            if len(calls) <= 2:
                raise httpx.ConnectError("synthetic outage")
            if len(calls) >= 4:
                stop.set()
            return object()

        with (
            patch.object(runner, "pulse", side_effect=flaky_pulse),
            patch.object(runner, "run_once", return_value=False),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            runner.run_until_stopped(self.client, 1, stop, poll_interval=0.01, presence_interval=0.01)
        self.assertGreaterEqual(len(calls), 4)

    def test_network_error_on_lease_renewal_is_retried_and_job_completes(self):
        outcomes = [httpx.ConnectError("synthetic outage")]

        def heartbeat(**_kwargs):
            value = outcomes.pop(0) if outcomes else object()
            if isinstance(value, Exception):
                raise value
            return value

        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", side_effect=heartbeat) as heartbeat_mock,
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.2) or {"provider": "fixture"}),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        self.assertGreaterEqual(heartbeat_mock.call_count, 2)
        complete.assert_called_once()
        fail.assert_not_called()

    def test_refused_lease_cancels_the_job_without_any_result(self):
        refusal = Error.from_dict({"error": "lease_not_owned"})
        started = time.monotonic()
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", return_value=refusal),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.3) or {"provider": "fixture"}),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        self.assertLess(time.monotonic() - started, 10)
        complete.assert_not_called()
        fail.assert_not_called()

    def _refused_long_job(self, step, task_input, started_work=lambda: True, voice_providers=None, **patches):
        """Run a job the server refuses to renew once its work has started; return its duration."""
        renewals = []
        refusals = {}

        def heartbeat(**_kwargs):
            renewals.append(1)
            if len(renewals) < 3 or not started_work():
                return object()
            refusals.setdefault("at", time.monotonic())
            return Error.from_dict({"error": "lease_not_owned"})

        self.task.step = step
        started = time.monotonic()
        with contextlib.ExitStack() as stack:
            stack.enter_context(patch.object(runner, "pulse", return_value=object()))
            stack.enter_context(patch.object(runner, "claim_job", return_value=self.claim))
            stack.enter_context(patch.object(runner, "get_job_input", return_value=task_input))
            stack.enter_context(patch.object(runner, "heartbeat_job", side_effect=heartbeat))
            self.complete_mock = stack.enter_context(patch.object(runner, "complete_job", return_value=object()))
            self.fail_mock = stack.enter_context(patch.object(runner, "fail_job", return_value=object()))
            for name, value in patches.items():
                stack.enter_context(patch.object(runner, name, value))
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.05, voice_providers=voice_providers))
        self.assertIn("at", refusals, "the server never refused the lease")
        # The fake work would last a minute: the job must end soon after the refusal, with margin for slow CI.
        self.stopped_after_refusal = time.monotonic() - refusals["at"]
        return time.monotonic() - started

    def _assert_cancelled_without_result(self, duration):
        self.assertLess(duration, 10)
        self.assertLess(self.stopped_after_refusal, 10)
        self.complete_mock.assert_not_called()
        self.fail_mock.assert_not_called()

    def test_refused_lease_aborts_a_long_remote_transcription(self):
        release = threading.Event()
        self.addCleanup(release.set)

        class HangingWhisper(BaseHTTPRequestHandler):
            def do_POST(self):
                self.rfile.read(int(self.headers["Content-Length"]))
                release.wait(60)  # a transcription far longer than the test budget
                self.send_response(500)
                self.end_headers()

            def log_message(self, *_args):
                pass

        server = ThreadingHTTPServer(("127.0.0.1", 0), HangingWhisper)
        server.daemon_threads = True
        threading.Thread(target=server.serve_forever, daemon=True).start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        transcriber = runner.OpenAICompatibleTranscriber(f"http://127.0.0.1:{server.server_port}", "synthetic-model")
        task_input = SimpleNamespace(pipeline={"vad": {"mode": "off"}}, vocabulary=[], transcription_model=None)
        duration = self._refused_long_job(
            "transcription.audio.0",
            task_input,
            download_media=lambda *_args: None,
            prepare_audio=lambda _src, wav, _index: _write_silence(wav),
            configured_transcriber=lambda *_args: transcriber,
        )
        self._assert_cancelled_without_result(duration)

    def test_refused_lease_kills_a_long_audio_preparation(self):
        with tempfile.TemporaryDirectory() as directory:
            ffmpeg = Path(directory) / "ffmpeg"
            ffmpeg.write_text("#!/bin/sh\necho $$ > \"$(dirname \"$0\")/pid\"\nexec sleep 60\n")
            ffmpeg.chmod(0o755)
            with patch.dict(os.environ, {"PATH": directory + os.pathsep + os.environ["PATH"]}):
                duration = self._refused_long_job(
                    "transcription.audio.0",
                    SimpleNamespace(pipeline=None, vocabulary=[], transcription_model=None),
                    started_work=(Path(directory) / "pid").exists,
                    download_media=lambda *_args: None,
                )
            pid = int((Path(directory) / "pid").read_text())
        self._assert_cancelled_without_result(duration)
        with self.assertRaises(ProcessLookupError):
            os.kill(pid, 0)

    def _voice_job(self, step, provider_name, provider, voice_extra=None):
        inference_started.clear()
        config = {"provider": provider_name, "model": "synthetic/model", "model_version": "v1", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}
        voice_input = {"config": config, **(voice_extra or {})}
        task_input = SimpleNamespace(voice=SimpleNamespace(to_dict=lambda: voice_input), duration_seconds=4.0)
        return self._refused_long_job(
            step,
            task_input,
            started_work=lambda: bool(inference_started),
            download_media=lambda *_args: None,
            prepare_audio=lambda _src, wav, _index: _write_silence(wav),
            voice_providers={provider_name: provider},
        )

    def test_refused_lease_interrupts_an_in_process_pyannote_diarization(self):
        with tempfile.TemporaryDirectory() as root:
            env = _model_env(root)
            pyannote = voice_pyannote.PyannoteProvider(environ=env, load_pipeline=lambda _p, _d: _long_inference, device_probe=lambda: "cpu")
            duration = self._voice_job("voice.diarization.0", "pyannote", pyannote)
        self._assert_cancelled_without_result(duration)
        self.assertFalse(pyannote._inference_lock.locked())

    def test_refused_lease_interrupts_an_in_process_nemo_diarization(self):
        with tempfile.TemporaryDirectory() as root:
            env = _model_env(root)
            nemo = voice_nemo.NemoProvider(environ=env, load_pipeline=lambda _p, _d: _long_inference, device_probe=lambda: "cpu")
            duration = self._voice_job("voice.diarization.0", "nemo", nemo)
        self._assert_cancelled_without_result(duration)
        self.assertFalse(nemo._inference_lock.locked())

    def test_refused_lease_stops_fingerprints_between_and_inside_embeddings(self):
        def open_audio(_path):
            return voice_pyannote.DecodedAudio(lambda start, end: _long_inference(None), 16000, 1, 16000 * 3600)

        turns = [{"start": float(i), "end": float(i) + 2.0} for i in range(0, 600, 3)]
        diarization = {"speakers": [{"label": "s0", "turns": turns}]}
        with tempfile.TemporaryDirectory() as root:
            env = _model_env(root)
            pyannote = voice_pyannote.PyannoteProvider(environ=env, load_embedder=lambda _p, _d: open_audio, device_probe=lambda: "cpu")
            duration = self._voice_job("voice.fingerprint.0", "pyannote", pyannote, {"diarization": diarization})
        self._assert_cancelled_without_result(duration)

    def test_refused_lease_stops_a_simulated_provider_at_its_next_cancellation_point(self):
        """A provider built on the shared embedding helper stops between two embeddings."""
        from snitch_worker import voice

        embedded = []

        class SimulatedProvider:
            def ensure_diarization_ready(self, _config):
                pass

            def ensure_ready(self, _config):
                pass

            def fingerprint(self, config, diarization, _audio, stream_index):
                def embed(start, _end):
                    embedded.append(start)
                    inference_started.append(1)
                    time.sleep(0.2)  # a short unit of work, returning to Python each time
                    return [1.0, 0.0]

                return voice.fingerprints_from_embedder(config, diarization, embed, stream_index, "m", "v1")

        turns = [{"index": i, "start": float(i * 3), "end": float(i * 3 + 2)} for i in range(400)]
        diarization = {"speakers": [{"label": "s0", "turns": turns}]}
        duration = self._voice_job("voice.fingerprint.0", "simulated", SimulatedProvider(), {"diarization": diarization})
        self._assert_cancelled_without_result(duration)
        self.assertLess(len(embedded), 40)  # 400 turns would take 80 s; the job stopped within seconds

    def test_result_is_discarded_when_the_lease_expired_during_an_outage(self):
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", side_effect=httpx.ConnectError("synthetic outage")),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.3) or {"provider": "fixture"}),
            patch.object(runner, "LEASE_TTL", 0.05),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        complete.assert_not_called()
        fail.assert_not_called()
