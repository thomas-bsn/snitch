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

from snitch_worker import runner


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


if __name__ == "__main__":
    unittest.main()
