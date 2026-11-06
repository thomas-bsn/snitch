import contextlib
import io
import json
import os
import ssl
import threading
import tempfile
import time
import unittest
import wave
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock, patch

import httpx
from snitch_client.client import AuthenticatedClient

from snitch_worker import log, runner


class FakeProviderTests(unittest.TestCase):
    def test_worker_override_is_sent_to_whisper_but_result_keeps_profile_identity(self):
        requests = []

        class Whisper(BaseHTTPRequestHandler):
            def do_POST(self):
                requests.append(self.rfile.read(int(self.headers["Content-Length"])))
                body = json.dumps({"model": "server-model", "language": "fr", "segments": []}).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *_args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Whisper)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, {
                "TRANSCRIPTION_API_URL": f"http://127.0.0.1:{server.server_port}",
                "TRANSCRIPTION_MODEL": "unused-env-model",
            }):
                path = str(Path(directory) / "fixture.wav")
                with wave.open(path, "wb") as audio:
                    audio.setnchannels(1)
                    audio.setsampwidth(2)
                    audio.setframerate(16000)
                    audio.writeframes(b"\0\0" * 1600)
                result = runner.transcribe_prepared_audio(
                    runner.configured_transcriber("library-model", "server-model"), path,
                    {"model": "library-model", "language": "fr", "vad": {"mode": "off"}}, [],
                )
                self.assertIn(b"server-model", requests[0])
                self.assertEqual(result["model"], "library-model")
        finally:
            server.shutdown()
            thread.join()
            server.server_close()

    def test_run_once_does_not_claim_after_stop_during_pulse(self):
        stop = threading.Event()

        def pulse_then_stop(**_kwargs):
            stop.set()
            return object()

        with (
            patch.object(runner, "pulse", side_effect=pulse_then_stop),
            patch.object(runner, "claim_job") as claim,
        ):
            self.assertFalse(runner.run_once(object(), stop=stop))

        claim.assert_not_called()

    def test_worker_pool_stops_claiming_and_drains_active_jobs(self):
        capacity = 3
        stop = threading.Event()
        release_jobs = threading.Event()
        started = threading.Event()
        lock = threading.Lock()
        active = 0
        peak_active = 0
        calls = 0

        def run_job(*_args, **_kwargs):
            nonlocal active, peak_active, calls
            with lock:
                calls += 1
                active += 1
                peak_active = max(peak_active, active)
                if active == capacity:
                    started.set()
            release_jobs.wait(2)
            with lock:
                active -= 1
            return True

        with patch.object(runner, "run_once", side_effect=run_job):
            thread = threading.Thread(
                target=runner.run_until_stopped,
                args=(object(), capacity, stop),
                kwargs={"poll_interval": 0.01},
            )
            thread.start()
            self.assertTrue(started.wait(2), "worker pool did not fill its capacity")
            stop.set()
            thread.join(0.05)
            self.assertTrue(thread.is_alive(), "shutdown did not wait for active jobs")
            release_jobs.set()
            thread.join(2)

        self.assertFalse(thread.is_alive(), "worker pool did not drain on shutdown")
        self.assertEqual(peak_active, capacity)
        self.assertEqual(calls, capacity, "worker claimed more jobs after shutdown")

    def test_pulse_declares_configured_machine_name_without_secrets(self):
        with (patch.dict(os.environ, {"SNITCH_WORKER_MACHINE_NAME": "Gaming PC"}),
              patch.object(runner, "pulse", return_value=object()) as pulse,
              patch("snitch_worker.declared_machine._hardware", return_value={"processor": "Fixture CPU", "memory_bytes": 8589934592, "gpu_name": "Fixture GPU", "gpu_memory_bytes": 4294967296, "gpu_driver": "555.1"}),
              patch.object(runner, "claim_job", return_value=SimpleNamespace(job=None))):
            self.assertFalse(runner.run_once(object(), capacity=1))
        machine = pulse.call_args.kwargs["body"].to_dict()["machine"]
        self.assertEqual(machine["name"], "Gaming PC")
        self.assertEqual(machine["worker_version"], "0.1.0")
        self.assertEqual(machine["gpu_name"], "Fixture GPU")
        self.assertEqual(machine["memory_bytes"], 8589934592)
        self.assertNotIn("HF_TOKEN", str(machine))

    def test_returns_deterministic_fixture_metadata_only(self):
        self.assertEqual(
            runner.fake_provider("fixture.echo"),
            {
                "provider": "fixture",
                "step": "fixture.echo",
            },
        )

    def test_fake_provider_rejects_non_fixture_steps(self):
        with self.assertRaises(ValueError):
            runner.fake_provider("transcription.whisper")

    def test_run_once_completes_fixture_without_requesting_media(self):
        http_client = SimpleNamespace(
            get=lambda *_args, **_kwargs: self.fail("media requested")
        )
        client = SimpleNamespace(
            _base_url="https://snitch.example",
            get_httpx_client=lambda: http_client,
        )
        task = SimpleNamespace(id=42, step="fixture.echo", input_version="v1")
        claim = SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")

        with (
            patch.object(runner, "pulse", return_value=object()) as pulse,
            patch.object(runner, "claim_job", return_value=claim),
            patch.object(runner, "heartbeat_job", return_value=object()),
            patch.object(
                runner,
                "get_job_input",
                return_value=SimpleNamespace(
                    to_dict=lambda: {"media_url": "/api/worker/jobs/42/media"}
                ),
            ),
            patch.object(runner, "complete_job", return_value=object()) as complete,
        ):
            self.assertTrue(runner.run_once(client, capacity=1))

        pulse.assert_called_once()
        advertised = pulse.call_args.kwargs["body"].to_dict()
        self.assertEqual(
            advertised["supported_steps"], ["fixture.*", "transcription.audio.*", "voice.*"]
        )
        complete.assert_called_once()
        result = complete.call_args.kwargs["body"].result.to_dict()
        self.assertEqual(result["provider"], "fixture")
        self.assertEqual(result["step"], "fixture.echo")
        self.assertEqual(
            complete.call_args.kwargs["x_worker_lease_token"],
            "lt_synthetic-lease-token",
        )

    def test_run_once_renews_lease_while_provider_is_working(self):
        client = SimpleNamespace(_base_url="https://snitch.example")
        task = SimpleNamespace(id=42, step="fixture.echo", input_version="v1")
        claim = SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=claim),
            patch.object(
                runner,
                "get_job_input",
                return_value=SimpleNamespace(
                    to_dict=lambda: {"media_url": "/api/worker/jobs/42/media"}
                ),
            ),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "heartbeat_job", return_value=object()) as heartbeat,
            patch.object(
                runner,
                "fake_provider",
                side_effect=lambda *_: time.sleep(0.05) or {"provider": "fixture"},
            ),
        ):
            self.assertTrue(runner.run_once(client, heartbeat_interval=0.01))

        self.assertGreaterEqual(heartbeat.call_count, 1)
        self.assertEqual(
            heartbeat.call_args.kwargs["x_worker_lease_token"],
            "lt_synthetic-lease-token",
        )
        self.assertEqual(
            complete.call_args.kwargs["x_worker_lease_token"],
            "lt_synthetic-lease-token",
        )

    def test_run_once_rejects_untrusted_https_certificate(self):
        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"id": "wk_fixture"}).encode())

            def log_message(self, *_args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Handler)
        fixture = Path(__file__).parent / "fixtures"
        tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        tls.load_cert_chain(
            fixture / "untrusted-test-cert.pem",
            fixture / "untrusted-test-key.pem",
        )
        server.socket = tls.wrap_socket(server.socket, server_side=True)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        self.addCleanup(thread.join, 2)
        client = AuthenticatedClient(
            base_url=f"https://localhost:{server.server_port}",
            token="rw_synthetic-test-token",
            verify_ssl=True,
        )
        self.addCleanup(client.get_httpx_client().close)

        with self.assertRaises(httpx.ConnectError):
            runner.run_once(client)

    def test_transcription_claim_prepares_global_stream_and_accepts_empty_result(self):
        media_request = SimpleNamespace(
            raise_for_status=lambda: None,
            iter_bytes=lambda: [b"synthetic-multitrack-media"],
        )

        class MediaStream:
            def __enter__(self):
                return media_request

            def __exit__(self, *_args):
                return False

        http_client = SimpleNamespace(stream=Mock(return_value=MediaStream()))
        client = SimpleNamespace(
            _base_url="https://snitch.example",
            get_httpx_client=lambda: http_client,
        )
        task = SimpleNamespace(
            id=84, step="transcription.audio.2", input_version="track-v1"
        )
        claim = SimpleNamespace(job=task, lease_token="lt_fixture-lease")
        captured = {}
        transcriber = SimpleNamespace(
            transcribe=lambda wav, vocabulary=None: captured.update(vocabulary=vocabulary) or {
                "provider": "fixture",
                "model": "deterministic-v1",
                "version": "deterministic-v1",
                "language": "fr",
                "segments": [],
            }
        )
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=claim),
            patch.object(runner, "heartbeat_job", return_value=object()),
            patch.object(
                runner,
                "get_job_input",
                return_value=SimpleNamespace(media_url="/api/worker/jobs/84/media", vocabulary=[SimpleNamespace(term="Final Fantasy XIV", variants=["FFXIV"], preferred_spelling="FFXIV")]),
            ),
            patch.object(runner, "prepare_audio") as prepare,
            patch.object(runner, "transcribe_prepared_audio", return_value={"provider":"fixture","model":"deterministic-v1","version":"deterministic-v1","language":"fr","segments":[]}) as transcribe,
            patch.object(runner, "complete_job", return_value=object()) as complete,
        ):
            self.assertTrue(runner.run_once(client, transcriber=transcriber))

        request = http_client.stream
        self.assertEqual(
            request.call_args.args[1],
            "https://snitch.example/api/worker/jobs/84/media",
        )
        self.assertEqual(
            request.call_args.kwargs["headers"]["X-Worker-Lease-Token"],
            "lt_fixture-lease",
        )
        self.assertEqual(prepare.call_args.args[2], 2)
        self.assertEqual(transcribe.call_args.args[3][0]["preferred_spelling"], "FFXIV")
        result = complete.call_args.kwargs["body"].result.to_dict()
        self.assertEqual(result["audio_stream_index"], 2)
        self.assertEqual(result["segments"], [])

    def test_provider_requires_explicit_endpoint_and_model(self):
        with self.assertRaises(ValueError):
            runner.configured_transcriber()

    def test_energy_vad_preserves_source_offsets_for_synthetic_audio(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "synthetic.wav"
            rate = 16000
            silence = b"\0\0" * (rate // 2)
            speech = b"\xff\x3f" * (rate // 2)
            with wave.open(str(path), "wb") as audio:
                audio.setnchannels(1)
                audio.setsampwidth(2)
                audio.setframerate(rate)
                audio.writeframes(silence + speech + silence + speech + silence)
            spans = runner.detect_voice_spans(str(path), {
                "vad": {"mode":"energy", "threshold":0.05, "min_speech_ms":100, "min_silence_ms":100, "padding_ms":50}
            })
            self.assertEqual(len(spans), 2)
            self.assertAlmostEqual(spans[0][0], 0.45, delta=0.03)
            self.assertAlmostEqual(spans[1][0], 1.45, delta=0.03)

    def test_run_once_completes_job_through_trusted_https_contract(self):
        state = {}
        job = {
            "id": 42,
            "clip_id": 19,
            "step": "fixture.echo",
            "input_version": "v1",
            "status": "leased",
            "priority": 0,
            "attempts": 1,
            "max_attempts": 3,
            "cycle": 1,
            "lease_owner": "wk_synthetic",
            "lease_expires_at": "2026-10-24T10:00:00Z",
            "next_attempt_at": "2026-10-24T10:00:00Z",
            "last_error": "",
            "created_at": "2026-10-24T09:00:00Z",
            "updated_at": "2026-10-24T09:00:00Z",
        }
        lease_token = "lt_synthetic-lease-token"

        class Handler(BaseHTTPRequestHandler):
            def _reply(self, status, data, content_type="application/json"):
                body = data if isinstance(data, bytes) else json.dumps(data).encode()
                self.send_response(status)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def _authorized(self):
                return self.headers.get("Authorization") == "Bearer rw_synthetic-token"

            def _lease_authorized(self):
                return self.headers.get("X-Worker-Lease-Token") == lease_token

            def do_POST(self):
                if not self._authorized():
                    return self._reply(401, {"error": "worker_token_invalid"})
                if self.path == "/api/worker/pulse":
                    pulse_body = json.loads(
                        self.rfile.read(int(self.headers["Content-Length"]))
                    )
                    state["pulse"] = pulse_body
                    return self._reply(
                        200,
                        {
                            "id": "wk_synthetic",
                            "name": "synthetic",
                            "max_concurrency": 1,
                            "reported_concurrency": 1,
                            "last_seen_at": None,
                            "created_at": "2026-10-24T09:00:00Z",
                            "revoked_at": None,
                            "revoke_reason": None,
                            "revocation_acknowledged_at": None,
                            "connected": False,
                        },
                    )
                if self.path == "/api/worker/jobs/claim":
                    return self._reply(200, {"job": job, "lease_token": lease_token})
                if self.path == "/api/worker/jobs/42/heartbeat":
                    if not self._lease_authorized():
                        return self._reply(409, {"error": "lease_not_owned"})
                    return self._reply(200, {"renewed": True})
                if self.path == "/api/worker/jobs/42/complete":
                    if not self._lease_authorized():
                        return self._reply(409, {"error": "lease_not_owned"})
                    request = json.loads(
                        self.rfile.read(int(self.headers["Content-Length"]))
                    )
                    state["result"] = request["result"]
                    return self._reply(
                        200, {**job, "status": "done", "result": state["result"]}
                    )
                return self._reply(404, {"error": "not_found"})

            def do_GET(self):
                if not self._authorized():
                    return self._reply(401, {"error": "worker_token_invalid"})
                if self.path == "/api/worker/jobs/42/input":
                    if not self._lease_authorized():
                        return self._reply(404, {"error": "job_not_found"})
                    return self._reply(
                        200,
                        {
                            "job": job,
                            "original_name": "synthetic.mp4",
                            "size_bytes": 123,
                            "video_codec": "",
                            "audio_codec": "",
                            "tracks": [],
                            "media_url": "/api/worker/jobs/42/media",
                            "vocabulary": [],
                            "pipeline": {"provider":"openai-compatible","model":"default","language":"fr","preparation":"wav-16khz-mono-v1","vad":{"mode":"off","version":"vad-off-v1"}},
                            "pipeline_version": "pipeline-v1",
                        },
                    )
                if self.path == "/api/worker/jobs/42/media":
                    self.fail("fake worker requested task media")
                return self._reply(404, {"error": "not_found"})

            def log_message(self, *_args):
                pass

        server = HTTPServer(("localhost", 0), Handler)
        fixture = Path(__file__).parent / "fixtures"
        tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        tls.load_cert_chain(
            fixture / "untrusted-test-cert.pem",
            fixture / "untrusted-test-key.pem",
        )
        server.socket = tls.wrap_socket(server.socket, server_side=True)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        self.addCleanup(thread.join, 2)
        client = AuthenticatedClient(
            base_url=f"https://localhost:{server.server_port}",
            token="rw_synthetic-token",
            verify_ssl=ssl.create_default_context(
                cafile=str(fixture / "untrusted-test-cert.pem")
            ),
        )
        self.addCleanup(client.get_httpx_client().close)

        self.assertTrue(runner.run_once(client))
        self.assertEqual(state["result"]["provider"], "fixture")
        self.assertEqual(state["result"]["step"], "fixture.echo")
        self.assertEqual(
            state["pulse"]["supported_steps"], ["fixture.*", "transcription.audio.*", "voice.*"]
        )


class RunnerLogTests(unittest.TestCase):
    LEASE = "lt_synthetic-lease-token"

    def setUp(self):
        self.stream = io.StringIO()
        self.addCleanup(log.configure, "INFO", io.StringIO())
        self.client = SimpleNamespace(_base_url="https://snitch.example")
        self.task = SimpleNamespace(
            id=42, clip_id=19, step="fixture.echo", input_version="v1", cycle=1, attempts=2, lease_owner="wk_synthetic"
        )
        self.claim = SimpleNamespace(job=self.task, lease_token=self.LEASE)
        self.shipped = []
        self.upload_fails = False
        self.shipped_at_failure = None

    def on_fail(self, **_kwargs):
        self.shipped_at_failure = [json.loads(line)["event"] for batch in self.shipped for line in batch]
        return object()

    def ship(self, client, id, body, x_worker_lease_token):
        self.assertEqual((id, x_worker_lease_token), (42, self.LEASE))
        if self.upload_fails:
            raise OSError("synthetic network failure")
        self.shipped.append(list(body.lines))
        return object()

    def lines(self):
        return [json.loads(line) for line in self.stream.getvalue().splitlines() if line.strip()]

    def run_job(self, level, complete=None, input_error=None, heartbeat_interval=45.0):
        log.configure(level, stream=self.stream)
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "heartbeat_job", return_value=object()),
            patch.object(runner, "get_job_input", side_effect=input_error, return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "complete_job", return_value=complete or object()),
            patch.object(runner, "fail_job", side_effect=self.on_fail),
            patch.object(runner, "append_job_logs", side_effect=self.ship),
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.05) or {"provider": "fixture"}),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=heartbeat_interval))

    def test_success_logs_transitions_with_the_claim_context_and_no_lease_token(self):
        self.run_job("INFO")
        events = {line["event"]: line for line in self.lines()}
        self.assertEqual(["job_claimed", "job_started", "job_completed"], list(events))
        for line in events.values():
            self.assertEqual(
                (line["clip_id"], line["job_id"], line["cycle"], line["attempt"], line["worker_id"], line["service"]),
                (19, 42, 1, 2, "wk_synthetic", "worker"),
            )
        self.assertIn("duration_ms", events["job_completed"])
        self.assertNotIn(self.LEASE, self.stream.getvalue())

    def test_failure_logs_the_code_without_the_raw_error_text(self):
        secret = "provider said token=hf_secret at /tmp/private/audio.wav"
        self.run_job("INFO", input_error=RuntimeError(secret))
        failed = [line for line in self.lines() if line["event"] == "job_failed"]
        self.assertEqual(len(failed), 1)
        self.assertEqual((failed[0]["error_code"], failed[0]["job_id"], failed[0]["attempt"]), ("provider_unavailable", 42, 2))
        self.assertEqual(failed[0]["level"], "WARN")
        for leaked in (secret, "hf_secret", "/tmp/private", self.LEASE):
            self.assertNotIn(leaked, self.stream.getvalue())

    def test_trapped_provider_and_sql_errors_leak_nothing_at_info_or_debug(self):
        for text in (
            "provider said token=hf_secret at https://api.example.test/v1?sig=abc",
            'duplicate key value violates unique constraint "users_email_key" Key (email)=(ann@example.test)',
        ):
            for level in ("INFO", "DEBUG"):
                self.stream.truncate(0)
                self.stream.seek(0)
                self.run_job(level, input_error=RuntimeError(text))
                output = self.stream.getvalue()
                self.assertIn("job_failed", output)
                for leaked in (text, "hf_secret", "sig=abc", "users_email_key", "ann@example.test", self.LEASE):
                    self.assertNotIn(leaked, output)

    def test_job_lines_are_shipped_masked_and_never_carry_secrets_or_clip_paths(self):
        secret = "provider said token=hf_secret at https://ann:pw-synth@api.example.test/v1?sig=abc for /tmp/private/audio.wav"
        self.run_job("DEBUG", input_error=RuntimeError(secret), heartbeat_interval=0.01)
        shipped = [json.loads(line) for batch in self.shipped for line in batch]
        events = [line["event"] for line in shipped]
        self.assertEqual(events[:2], ["job_claimed", "job_started"])
        self.assertIn("job_failed", events)
        self.assertTrue(all(len(batch) <= 50 for batch in self.shipped))
        text = json.dumps(shipped)
        for leaked in (secret, "hf_secret", "ann:pw-synth", "sig=abc", "/tmp/private", "audio.wav", self.LEASE, "Bearer"):
            self.assertNotIn(leaked, text)

    def test_failure_lines_are_sent_before_the_failure_is_reported(self):
        self.run_job("INFO", input_error=RuntimeError("synthetic"))
        self.assertIn("job_failed", self.shipped_at_failure)

    def test_a_failing_log_upload_never_fails_the_job(self):
        self.upload_fails = True
        self.run_job("INFO")
        self.assertIn("job_completed", [line["event"] for line in self.lines()])

    def test_heartbeats_and_empty_claims_only_appear_at_debug(self):
        self.run_job("INFO", heartbeat_interval=0.01)
        self.assertNotIn("job_heartbeat", self.stream.getvalue())
        self.stream.truncate(0)
        self.stream.seek(0)
        self.run_job("DEBUG", heartbeat_interval=0.01)
        self.assertIn("job_heartbeat", [line["event"] for line in self.lines()])
        self.assertNotIn(self.LEASE, self.stream.getvalue())
        log.configure("INFO", stream=self.stream)
        with patch.object(runner, "pulse", return_value=object()), patch.object(runner, "claim_job", return_value=SimpleNamespace(job=None)):
            self.assertFalse(runner.run_once(self.client))
        self.assertNotIn("claim_empty", self.stream.getvalue())


class WhisperFailureTests(unittest.TestCase):
    """A failing Whisper call is reported with its own cause and a typed diagnostic, never with its text."""

    LEASE = "lt_synthetic-lease-token"
    PRIVATE = "synthetic transcript words token=hf_secret /tmp/private/audio.wav"

    def run_failure(self, status=None, body=b"{}", post_error=None):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        sent = []

        class Whisper(BaseHTTPRequestHandler):
            def do_POST(self):
                self.rfile.read(int(self.headers["Content-Length"]))
                self.send_response(status or 200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *_args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Whisper)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        task = SimpleNamespace(id=7, clip_id=3, step="transcription.audio.0", input_version="v1", cycle=0, attempts=1, lease_owner="wk_synthetic")
        task_input = SimpleNamespace(media_url="/media", pipeline=None, transcription_model=None, vocabulary=[])

        def prepare(_media, wav_path, _index):
            with wave.open(wav_path, "wb") as audio:
                audio.setnchannels(1)
                audio.setsampwidth(2)
                audio.setframerate(16000)
                audio.writeframes(b"\0\0" * 1600)

        def capture(**kwargs):
            sent.append(kwargs["body"])
            return object()

        try:
            transcriber = runner.OpenAICompatibleTranscriber(f"http://127.0.0.1:{server.server_port}", "whisper-synthetic")
            patches = [
                patch.object(runner, "pulse", return_value=object()),
                patch.object(runner, "claim_job", return_value=SimpleNamespace(job=task, lease_token=self.LEASE)),
                patch.object(runner, "heartbeat_job", return_value=object()),
                patch.object(runner, "get_job_input", return_value=task_input),
                patch.object(runner, "download_media"),
                patch.object(runner, "prepare_audio", side_effect=prepare),
                patch.object(runner, "fail_job", side_effect=capture),
            ]
            if post_error is not None:
                patches.append(patch.object(runner.httpx.Client, "post", side_effect=post_error))
            with contextlib.ExitStack() as stack:
                for active in patches:
                    stack.enter_context(active)
                self.assertTrue(runner.run_once(SimpleNamespace(_base_url="https://snitch.example"), transcriber=transcriber))
        finally:
            server.shutdown()
            thread.join()
            server.server_close()
        self.assertEqual(len(sent), 1)
        for leaked in (self.PRIVATE, "hf_secret", "/tmp/private", self.LEASE):
            self.assertNotIn(leaked, stream.getvalue())
            self.assertNotIn(leaked, json.dumps(sent[0].to_dict()))
        return sent[0].to_dict()

    def test_each_whisper_failure_family_has_its_own_code_and_diagnostic(self):
        for name, expected, extra in (
            ("network", "whisper_network", {"post_error": httpx.ConnectError(self.PRIVATE)}),
            ("timeout", "whisper_timeout", {"post_error": httpx.ReadTimeout(self.PRIVATE)}),
            ("http", "whisper_http_error", {"status": 503, "body": self.PRIVATE.encode()}),
            ("missing model", "whisper_model_missing", {"status": 404, "body": self.PRIVATE.encode()}),
            ("invalid response", "whisper_invalid_response", {"body": self.PRIVATE.encode()}),
            ("no segments", "whisper_invalid_response", {"body": b'{"language":"fr"}'}),
        ):
            with self.subTest(name):
                payload = self.run_failure(**extra)
                self.assertEqual(payload["error"], expected)
                self.assertEqual((payload["provider"], payload["model"]), ("openai-compatible", "whisper-synthetic"))
                self.assertGreaterEqual(payload["duration_ms"], 0)
                if "status" in extra:
                    self.assertEqual(payload["http_status"], extra["status"])
                else:
                    self.assertNotIn("http_status", payload)

    def test_a_whisper_http_failure_logs_a_cleaned_bounded_request_line(self):
        # ADR 0028: method, URL without credentials, HTTP status, latency and a bounded excerpt, no secret.
        body = (b'{"error":"overloaded Bearer abc.def hf_SyntheticToken123 /tmp/private/audio.wav '
                + b"x" * 600 + b'"}')
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        transcriber = runner.OpenAICompatibleTranscriber("http://ann:pw-synth@127.0.0.1:9", "whisper-synthetic")
        request = httpx.Request("POST", "http://ann:pw-synth@127.0.0.1:9/v1/audio/transcriptions")
        response = httpx.Response(503, content=body, request=request)
        failure = httpx.HTTPStatusError("x", request=request, response=response)
        with patch.object(runner.OpenAICompatibleTranscriber, "_transcribe", side_effect=failure):
            with self.assertRaises(runner.TranscriptionError) as caught:
                transcriber.transcribe("unused.wav")
        runner.log_whisper_failure(caught.exception, 12)
        [line] = [json.loads(row) for row in stream.getvalue().splitlines() if "whisper_request_failed" in row]
        self.assertEqual((line["method"], line["http_status"], line["latency_ms"]), ("POST", 503, 12))
        self.assertEqual(line["code"], "whisper_http_error")
        self.assertLessEqual(len(line["excerpt"]), 240)
        self.assertIn("overloaded", line["excerpt"])
        self.assertTrue(line["url"].startswith("http://127.0.0.1:9"))
        for leaked in ("abc.def", "hf_SyntheticToken123", "/tmp/private", "ann:pw-synth", "audio.wav"):
            self.assertNotIn(leaked, stream.getvalue())

    def test_other_failures_keep_the_generic_provider_code_without_diagnostic(self):
        payload = self.run_failure(post_error=RuntimeError(self.PRIVATE))
        self.assertEqual(payload["error"], "provider_unavailable")
        self.assertNotIn("provider", payload)


if __name__ == "__main__":
    unittest.main()


class ModelsLoopTests(unittest.TestCase):
    """Orders are pulled on their own thread, so a long job never leaves one pending (#327)."""

    def test_orders_are_pulled_while_no_claiming_loop_is_free(self):
        from unittest.mock import patch

        stop = threading.Event()
        pulls = []

        def pull(client):
            pulls.append(client)
            if len(pulls) >= 2:
                stop.set()

        with patch.object(runner, "sync_models_once", pull), patch.object(runner, "AuthenticatedClient", object):
            runner._models_loop(object(), stop, 0.01)
        self.assertGreaterEqual(len(pulls), 2)

    def test_a_failed_pull_does_not_stop_the_loop(self):
        from unittest.mock import patch

        calls = {"n": 0}

        def failing(*_args, **_kwargs):
            calls["n"] += 1
            raise RuntimeError("down")

        with patch.object(runner.voice_models, "sync_voice_models", failing), patch.object(runner.model_orders, "sync_orders", failing):
            runner.sync_models_once(object())
        self.assertEqual(calls["n"], 2)  # each step runs on its own

    def test_run_once_no_longer_pulls_orders(self):
        import inspect

        self.assertNotIn("sync_orders", inspect.getsource(runner.run_once))
