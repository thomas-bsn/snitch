"""A redeploy can return a proxy page before the API becomes available again."""
import io
import threading
import unittest
from types import SimpleNamespace
from unittest.mock import patch

import httpx

from snitch_worker import log, runner


class ReconnectTests(unittest.TestCase):
    def setUp(self):
        log.configure("INFO", io.StringIO())

    def test_claiming_resumes_after_a_proxy_page_without_restarting_worker(self):
        for status, body in [(502, "<html>Bad Gateway</html>"), (503, "{\"detail\":\"restarting\"}"), (200, "<html>Starting</html>"), (503, "null")]:
            with self.subTest(status=status, body=body):
                stop = threading.Event()
                calls = []

                def serve(request):
                    calls.append(request.url.path)
                    if len(calls) == 1:
                        return httpx.Response(status, text=body)
                    stop.set()
                    return httpx.Response(200, json={"job": None, "lease_token": ""})

                with httpx.Client(base_url="https://snitch.example", transport=httpx.MockTransport(serve)) as http:
                    client = SimpleNamespace(get_httpx_client=lambda: http)
                    timer = threading.Timer(0.5, stop.set)
                    timer.start()
                    try:
                        with patch.object(runner, "_send_pulse"):
                            runner._worker_loop(client, 1, stop, 0.001, 0.01)
                    finally:
                        stop.set()
                        timer.cancel()
                self.assertGreaterEqual(len(calls), 2, "the worker must keep claiming after the API returns")

    def test_retry_delay_stays_bounded_after_hours_offline(self):
        self.assertEqual(runner.presence_retry_delay(100000), runner.PRESENCE_RETRY_CAP)

    def test_lease_renewal_recovers_from_a_proxy_page(self):
        stop = threading.Event()
        lease = runner.LeaseWatch()
        calls = []

        def serve(request):
            calls.append(request.url.path)
            if len(calls) == 1:
                return httpx.Response(502, text="<html>Bad Gateway</html>")
            stop.set()
            return httpx.Response(200, json={"status": "ok"})

        with httpx.Client(base_url="https://snitch.example", transport=httpx.MockTransport(serve)) as http:
            client = SimpleNamespace(get_httpx_client=lambda: http)
            timer = threading.Timer(0.5, stop.set)
            timer.start()
            try:
                with patch.object(runner, "PRESENCE_RETRY_CAP", 0.001):
                    runner._heartbeat_loop(client, 7, "synthetic-lease", 0.001, stop, lease)
            finally:
                stop.set()
                timer.cancel()
        self.assertFalse(lease.lost.is_set())
        self.assertGreaterEqual(len(calls), 2)
