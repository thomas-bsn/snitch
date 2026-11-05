"""Issue #321: a Worker already Online keeps claiming after Snitch is redeployed, without a restart.

The fake Snitch below is a real HTTP/1.1 server (keep-alive connections, so the Worker reuses its sockets) that can
go away and come back on the same port, answer a proxy page, hang, or answer JSON the generated client cannot read.
Synthetic data only: no media, a fixture step.
"""
import io
import json
import os
import socket
import tempfile
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from unittest.mock import patch

import httpx
from snitch_client.client import AuthenticatedClient
from snitch_client.models.worker_claim import WorkerClaim

from snitch_worker import cancellation, log, runner

TOKEN = "rw_synthetic-worker-token-321"
LEASE = "lt_synthetic-lease-token-321"
STAMP = "2026-11-05T00:00:00+00:00"
WORKER = {
    "id": "wk_synthetic_worker_identity_0001", "name": "worker-4", "max_concurrency": 1, "reported_concurrency": 1,
    "last_seen_at": STAMP, "created_at": STAMP, "revoked_at": None, "revoke_reason": "",
    "revocation_acknowledged_at": None, "connected": True,
}


def _job(status):
    return {
        "id": 213, "clip_id": 5, "step": "fixture.echo", "input_version": "v1", "status": status, "priority": 0,
        "attempts": 0 if status == "pending" else 1, "max_attempts": 3, "lease_owner": None if status == "pending" else WORKER["id"],
        "lease_expires_at": None, "next_attempt_at": STAMP, "last_error": "", "created_at": STAMP, "updated_at": STAMP, "cycle": 1,
    }


class FakeSnitch:
    """Modes: up, down (listener and open connections gone), proxy (502 page), html (200 page), hang, garbled."""

    def __init__(self):
        self.mode = "up"
        self.job_pending = False
        self.lock = threading.Lock()
        self.claims = 0
        self.leases = []
        self.completed = []
        self.tokens = set()
        self.connections = set()
        self.port = 0
        self.server = None
        self.start()

    def start(self):
        fake = self

        class Handler(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def log_message(self, *_args):
                pass

            def setup(self):
                super().setup()
                with fake.lock:
                    fake.connections.add(self.connection)

            def finish(self):
                with fake.lock:
                    fake.connections.discard(self.connection)
                try:
                    super().finish()
                except OSError:
                    pass

            def reply(self, status, body, ctype="application/json"):
                data = body.encode() if isinstance(body, str) else json.dumps(body).encode()
                self.send_response(status)
                self.send_header("Content-Type", ctype)
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)

            def handle_any(self):
                length = int(self.headers.get("Content-Length") or 0)
                if length:
                    self.rfile.read(length)
                fake.tokens.add(self.headers.get("Authorization", ""))
                mode = fake.mode
                if mode == "proxy":
                    return self.reply(502, "<html><body>502 Bad Gateway</body></html>", "text/html")
                if mode == "html":
                    return self.reply(200, "<html><body>Snitch is starting</body></html>", "text/html")
                if mode == "hang":
                    time.sleep(1.0)
                    return self.reply(503, "<html>timeout</html>", "text/html")
                if mode == "garbled":
                    return self.reply(200, [])
                return self.route()

            def route(self):
                path = self.path
                if path == "/api/worker/pulse":
                    return self.reply(200, WORKER)
                if path == "/api/worker/transcription-config":
                    return self.reply(200, {"mode": "unconfigured", "version": ""})
                if path == "/api/worker/hf-token/version":
                    return self.reply(200, {"version": "0"})
                if path == "/api/worker/voice-models":
                    return self.reply(200, {"models": [], "orders": []})
                if path == "/api/worker/jobs/claim":
                    with fake.lock:
                        fake.claims += 1
                        if fake.job_pending:
                            fake.job_pending = False
                            fake.leases.append(213)
                            return self.reply(200, {"job": _job("leased"), "lease_token": LEASE})
                    return self.reply(200, {"job": None, "lease_token": ""})
                if path == "/api/worker/jobs/213/input":
                    return self.reply(200, {"job": _job("leased"), "original_name": "fixture", "size_bytes": 1, "video_codec": "", "audio_codec": "", "tracks": [], "media_url": "/api/worker/jobs/213/media", "vocabulary": []})
                if path == "/api/worker/jobs/213/heartbeat":
                    return self.reply(200, _job("leased"))
                if path == "/api/worker/jobs/213/logs":
                    return self.reply(200, {"accepted": 1})
                if path == "/api/worker/jobs/213/complete":
                    with fake.lock:
                        fake.completed.append(213)
                    return self.reply(200, _job("done"))
                return self.reply(404, {"error": "not_found"})

            do_GET = do_POST = handle_any

        self.server = ThreadingHTTPServer(("127.0.0.1", self.port), Handler)
        self.server.daemon_threads = True
        self.port = self.server.server_address[1]
        threading.Thread(target=self.server.serve_forever, daemon=True).start()

    def go_down(self):
        """The old Snitch is gone: no listener, and the Worker's kept-alive sockets are reset."""
        self.mode = "down"
        self.server.shutdown()
        self.server.server_close()
        with self.lock:
            for connection in list(self.connections):
                try:
                    connection.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass

    def come_back(self, mode="up"):
        self.mode = mode
        self.start()

    def close(self):
        self.server.shutdown()
        self.server.server_close()


def wait_until(predicate, timeout=10.0):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return True
        time.sleep(0.02)
    return predicate()


def lines(stream):
    return [json.loads(line) for line in stream.getvalue().splitlines() if line.strip()]


class RedeployRecoveryTests(unittest.TestCase):
    def setUp(self):
        self.logs = io.StringIO()
        log.configure("INFO", self.logs)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        environ = patch.dict(os.environ, {
            "SNITCH_WORKER_MODELS_DIR": self.directory.name,
            "SNITCH_HF_TOKEN_FILE": os.path.join(self.directory.name, "hf", "token"),
            "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.directory.name, "embedding"),
            "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(self.directory.name, "diarization"),
        })
        environ.start()
        self.addCleanup(environ.stop)
        for name, value in (("PRESENCE_RETRY_CAP", 0.05), ("CLAIM_LOG_INTERVAL", 60.0)):
            patcher = patch.object(runner, name, value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def start_worker(self, snitch):
        client = AuthenticatedClient(base_url=f"http://127.0.0.1:{snitch.port}", token=TOKEN, timeout=httpx.Timeout(0.3))
        stop = threading.Event()
        thread = threading.Thread(
            target=runner.run_until_stopped, args=(client, 1, stop),
            kwargs={"poll_interval": 0.05, "presence_interval": 0.05, "models_dir": self.directory.name}, daemon=True,
        )
        thread.start()

        def shutdown():
            stop.set()
            thread.join(5)
            client.get_httpx_client().close()

        self.addCleanup(shutdown)
        return thread

    def test_a_pending_job_is_claimed_and_done_after_a_redeploy_without_restarting_the_worker(self):
        snitch = FakeSnitch()
        self.addCleanup(snitch.close)
        thread = self.start_worker(snitch)
        self.assertTrue(wait_until(lambda: snitch.claims >= 2), "the Worker never claimed before the redeploy")

        # Redeploy: Snitch disappears (sockets reset), the proxy answers pages, the API hangs, then answers
        # JSON the generated client cannot read, and only then is healthy again, with job #213 ready.
        snitch.go_down()
        time.sleep(0.4)
        for mode in ("proxy", "html", "hang", "garbled"):
            if mode == "proxy":
                snitch.come_back("proxy")
            else:
                snitch.mode = mode
            time.sleep(0.5)
        self.assertEqual(snitch.leases, [], "nothing may be leased while Snitch is unhealthy")
        snitch.job_pending = True
        snitch.mode = "up"

        self.assertTrue(wait_until(lambda: snitch.completed == [213]), "job #213 was not done after Snitch came back")
        self.assertTrue(thread.is_alive())
        self.assertEqual(snitch.leases, [213], "the job must be leased exactly once")
        self.assertEqual(snitch.tokens, {f"Bearer {TOKEN}"}, "the same Worker identity must be used throughout")

        events = lines(self.logs)
        failures = [e for e in events if e["event"] == "claim_poll_failed"]
        self.assertTrue(failures, "the claim failures must be visible")
        self.assertLessEqual(len(failures), 3, "failures must be rate limited, not one line per attempt")
        self.assertTrue(all(e.get("operation") and e.get("error_category") for e in failures), failures)
        self.assertTrue(any(e["event"] == "claim_poll_recovered" for e in events))
        self.assertNotIn(TOKEN, self.logs.getvalue())
        self.assertNotIn(LEASE, self.logs.getvalue())


class ClaimLoopResilienceTests(unittest.TestCase):
    def setUp(self):
        self.logs = io.StringIO()
        log.configure("INFO", self.logs)
        self.addCleanup(log.configure, "INFO", io.StringIO())

    def test_an_unexpected_error_does_not_end_the_claim_loop(self):
        stop = threading.Event()
        calls = []

        def run_once(*_args, **_kwargs):
            calls.append(1)
            if len(calls) == 1:
                raise IndexError("synthetic bug outside the recoverable list")
            if len(calls) == 2:
                raise cancellation.JobCancelled()
            stop.set()
            return False

        with patch.object(runner, "run_once", side_effect=run_once), patch.object(runner, "PRESENCE_RETRY_CAP", 0.01):
            runner._worker_loop(object(), 1, stop, 0.001, 0.01)
        self.assertEqual(len(calls), 3)
        failures = [e for e in lines(self.logs) if e["event"] == "claim_poll_failed"]
        self.assertEqual(failures[0]["error_category"], "internal")
        self.assertEqual(failures[0]["error_type"], "IndexError")

    def test_failures_back_off_without_a_hot_loop(self):
        stop = threading.Event()
        stamps = []

        def failing(*_args, **_kwargs):
            stamps.append(time.monotonic())
            if len(stamps) >= 4:
                stop.set()
            raise httpx.ConnectError("synthetic outage")

        with patch.object(runner, "run_once", side_effect=failing), patch.object(runner, "PRESENCE_RETRY_CAP", 0.2):
            runner._worker_loop(object(), 1, stop, 0.01, 0.01)
        gaps = [later - earlier for earlier, later in zip(stamps, stamps[1:])]
        self.assertGreaterEqual(min(gaps), 0.009)
        self.assertGreater(gaps[-1], gaps[0], "consecutive failures must wait longer")
        self.assertLess(max(gaps), 0.5, "the wait is bounded")

    def test_a_claim_loop_that_dies_is_reported_and_restarted(self):
        stop = threading.Event()
        calls = []

        class Fatal(BaseException):
            pass

        def run_once(*_args, **_kwargs):
            calls.append(1)
            if len(calls) == 1:
                raise Fatal()
            stop.set()
            return False

        with (
            patch.object(runner, "run_once", side_effect=run_once),
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            runner.run_until_stopped(object(), 1, stop, poll_interval=0.01, presence_interval=0.01, models_dir=tempfile.gettempdir())
        self.assertEqual(len(calls), 2, "the claim loop must be restarted after it died")
        stopped = [e for e in lines(self.logs) if e["event"] == "claim_loop_stopped"]
        self.assertEqual(len(stopped), 1)
        self.assertEqual(stopped[0]["error_type"], "Fatal")

    def test_a_failing_pulse_does_not_keep_the_worker_from_claiming(self):
        claims = []

        def refuse_pulse(**_kwargs):
            raise httpx.HTTPStatusError("400", request=httpx.Request("POST", "https://snitch.example"), response=httpx.Response(400))

        def claim(**_kwargs):
            claims.append(1)
            return WorkerClaim(job=None, lease_token="")

        with patch.object(runner, "pulse", side_effect=refuse_pulse), patch.object(runner, "claim_job", side_effect=claim):
            self.assertFalse(runner.run_once(object()))
        self.assertEqual(claims, [1], "presence keeps the canonical pulse; the claim does not depend on it")

    def test_an_unexpected_job_error_stops_the_lease_renewal_and_records_the_failure(self):
        renewals, failures = [], []
        task = type("Task", (), {"id": 7, "clip_id": 3, "step": "fixture.echo", "input_version": "v1", "cycle": 1, "attempts": 1, "lease_owner": "wk"})()
        claim = type("Claim", (), {"job": task, "lease_token": LEASE})()

        def heartbeat(**_kwargs):
            renewals.append(time.monotonic())
            return object()

        with (
            patch.object(runner, "get_job_input", return_value=object()),
            patch.object(runner, "fake_provider", side_effect=IndexError("synthetic provider bug")),
            patch.object(runner, "heartbeat_job", side_effect=heartbeat),
            patch.object(runner, "fail_job", side_effect=lambda **kwargs: failures.append(kwargs["body"].error) or object()),
        ):
            shipper = log.LineShipper(7, lambda _lines: None)
            try:
                self.assertTrue(runner._run_claimed(object(), claim, 0.01, None, None, shipper))
            finally:
                shipper.close()
        count = len(renewals)
        time.sleep(0.1)
        self.assertEqual(len(renewals), count, "the lease renewal must stop with the job")
        self.assertEqual(failures, ["provider_unavailable"])


if __name__ == "__main__":
    unittest.main()
