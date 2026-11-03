import io
import json
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from types import SimpleNamespace
from unittest.mock import patch

import httpx

from snitch_worker import log, runner
from snitch_worker.transcription_config import ReachabilityMonitor


class _Server:
    """Local synthetic Whisper-like endpoint whose status and body are chosen by the test."""

    def __init__(self, status=200, body=None):
        outer = self
        self.status = status
        self.body = body if body is not None else json.dumps({"data": [{"id": "synthetic-model"}]}).encode()

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(outer.status)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(outer.body)))
                self.end_headers()
                self.wfile.write(outer.body)

            def log_message(self, *_args):
                pass

        self.httpd = HTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)

    def __enter__(self):
        self.thread.start()
        return f"http://127.0.0.1:{self.httpd.server_port}"

    def __exit__(self, *_args):
        self.httpd.shutdown()
        self.thread.join()
        self.httpd.server_close()


class ReachabilityProbeTests(unittest.TestCase):
    def test_reachable_endpoint_reports_latency_and_no_failure(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        with _Server() as endpoint:
            monitor.set_target(endpoint, "synthetic-model")
            monitor.probe_once()
        snapshot = monitor.snapshot()
        self.assertTrue(snapshot["reachable"])
        self.assertIsInstance(snapshot["latency_ms"], int)
        self.assertIsNone(snapshot["last_failure"])
        self.assertIn("checked_at", snapshot)

    def test_missing_model_still_counts_as_reachable(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        with _Server() as endpoint:
            monitor.set_target(endpoint, "other-model")
            monitor.probe_once()
        self.assertTrue(monitor.snapshot()["reachable"])

    def test_http_failure_keeps_sanitized_bounded_details(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        long_body = ("x" * 500).encode()
        with _Server(status=503, body=long_body) as endpoint:
            monitor.set_target(endpoint.replace("http://", "http://user:synthetic-secret@") + "/base?token=synthetic", "synthetic-model")
            monitor.probe_once()
        snapshot = monitor.snapshot()
        self.assertFalse(snapshot["reachable"])
        failure = snapshot["last_failure"]
        self.assertEqual(failure["method"], "GET")
        self.assertEqual(failure["status_code"], 503)
        self.assertEqual(failure["error_type"], "http_status")
        self.assertTrue(failure["url"].endswith("/base/v1/models"))
        self.assertNotIn("synthetic-secret", failure["url"])
        self.assertNotIn("token", failure["url"])
        self.assertNotIn("@", failure["url"])
        self.assertLessEqual(len(failure["body_excerpt"]), 200)
        self.assertIsInstance(failure["latency_ms"], int)
        self.assertIn("failed_at", failure)

    def test_connection_refused_is_a_connection_failure_without_status(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        monitor.set_target("http://127.0.0.1:1", "synthetic-model")
        monitor.probe_once()
        failure = monitor.snapshot()["last_failure"]
        self.assertEqual(failure["error_type"], "connection")
        self.assertIsNone(failure["status_code"])
        self.assertEqual(failure["url"], "http://127.0.0.1:1/v1/models")

    def test_timeout_is_reported_as_timeout(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        monitor.set_target("http://127.0.0.1:9", "synthetic-model")
        with patch("snitch_worker.transcription_config.httpx.get", side_effect=httpx.ReadTimeout("synthetic")):
            monitor.probe_once()
        self.assertEqual(monitor.snapshot()["last_failure"]["error_type"], "timeout")

    def test_last_failure_survives_a_later_success(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        monitor.set_target("http://127.0.0.1:1", "synthetic-model")
        monitor.probe_once()
        with _Server() as endpoint:
            monitor.set_target(endpoint, "synthetic-model")
            monitor.probe_once()
        snapshot = monitor.snapshot()
        self.assertTrue(snapshot["reachable"])
        self.assertIsNone(snapshot["last_failure"])

    def test_no_snapshot_without_target(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        monitor.probe_once()
        self.assertIsNone(monitor.snapshot())


class ReachabilityScheduleTests(unittest.TestCase):
    def setUp(self):
        self.addCleanup(log.configure, "INFO", io.StringIO())
        log.configure("INFO", io.StringIO())

    def test_probe_runs_periodically_on_its_own_thread(self):
        monitor = ReachabilityMonitor(interval=0.02, timeout=5)
        monitor.set_target("http://127.0.0.1:1", "synthetic-model")
        stop = threading.Event()
        calls = []

        def counted(*_args, **_kwargs):
            calls.append(threading.current_thread().name)
            if len(calls) >= 3:
                stop.set()

        with patch.object(monitor, "probe_once", side_effect=counted):
            thread = threading.Thread(target=monitor.run, args=(stop,), name="snitch-reachability")
            thread.start()
            thread.join(5)
        self.assertFalse(thread.is_alive())
        self.assertGreaterEqual(len(calls), 3)
        self.assertTrue(all(name == "snitch-reachability" for name in calls))

    def test_slow_probe_never_delays_presence_pulses(self):
        monitor = ReachabilityMonitor(interval=0.001, timeout=5)
        monitor.set_target("http://127.0.0.1:1", "synthetic-model")
        release = threading.Event()
        stop = threading.Event()
        pulse_times = []

        def slow_probe():
            release.wait(2)

        def record_pulse(**_kwargs):
            pulse_times.append(time.monotonic())
            if len(pulse_times) >= 5:
                stop.set()
            return object()

        with (
            patch.object(monitor, "probe_once", side_effect=slow_probe),
            patch.object(runner, "pulse", side_effect=record_pulse),
            patch.object(runner, "run_once", return_value=False),
        ):
            started = time.monotonic()
            runner.run_until_stopped(
                SimpleNamespace(_base_url="https://snitch.example"),
                1,
                stop,
                poll_interval=0.01,
                presence_interval=0.01,
                reachability=monitor,
            )
            release.set()
        self.assertGreaterEqual(len(pulse_times), 5)
        self.assertLess(pulse_times[4] - started, 1.5)

    def test_pulse_carries_the_latest_reachability_result(self):
        monitor = ReachabilityMonitor(interval=60, timeout=5)
        monitor.set_target("http://127.0.0.1:1", "synthetic-model")
        monitor.probe_once()
        with patch.object(runner, "pulse", return_value=object()) as pulse:
            runner._send_pulse(SimpleNamespace(), 1, monitor)
        body = pulse.call_args.kwargs["body"].to_dict()
        reachability = body["transcription_reachability"]
        self.assertFalse(reachability["reachable"])
        self.assertEqual(reachability["last_failure"]["error_type"], "connection")
        self.assertEqual(reachability["last_failure"]["method"], "GET")

    def test_pulse_omits_reachability_before_any_target_is_known(self):
        with patch.object(runner, "pulse", return_value=object()) as pulse:
            runner._send_pulse(SimpleNamespace(), 1, ReachabilityMonitor(interval=60, timeout=5))
        self.assertNotIn("transcription_reachability", pulse.call_args.kwargs["body"].to_dict())


if __name__ == "__main__":
    unittest.main()
