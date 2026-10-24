import hashlib
import json
import ssl
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import httpx
from receipts_api.client import AuthenticatedClient

from receipts_worker import runner


class FakeProviderTests(unittest.TestCase):
    def test_returns_deterministic_fixture_metadata_only(self):
        media = b"synthetic-video-fixture"

        self.assertEqual(
            runner.fake_provider(media, "fixture.echo"),
            {
                "provider": "fixture",
                "step": "fixture.echo",
                "media_sha256": hashlib.sha256(media).hexdigest(),
            },
        )

    def test_run_once_polls_fetches_scoped_media_and_completes(self):
        media = b"synthetic-video-fixture"
        http_client = SimpleNamespace(
            get=lambda url: self._media_response(url, media)
        )
        client = SimpleNamespace(
            _base_url="https://receipts.example",
            get_httpx_client=lambda: http_client,
        )
        task = SimpleNamespace(id=42, step="fixture.echo", input_version="v1")
        claim = SimpleNamespace(job=task)

        with (
            patch.object(runner, "pulse", return_value=object()) as pulse,
            patch.object(runner, "claim_job", return_value=claim),
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
        complete.assert_called_once()
        result = complete.call_args.kwargs["body"].result.to_dict()
        self.assertEqual(result["provider"], "fixture")
        self.assertEqual(result["media_sha256"], hashlib.sha256(media).hexdigest())

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

    @staticmethod
    def _media_response(url, media):
        if url != "https://receipts.example/api/worker/jobs/42/media":
            raise AssertionError(f"unexpected media URL: {url}")
        return SimpleNamespace(content=media, raise_for_status=lambda: None)


if __name__ == "__main__":
    unittest.main()
