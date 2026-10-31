import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from unittest.mock import Mock, patch, MagicMock

from snitch_worker.transcription_config import test_configuration as _test_configuration, sync_configuration, active_configuration, activate, ActiveConfiguration


class ConfigurationTests(unittest.TestCase):
    def test_worker_distinguishes_network_failure_from_missing_model(self):
        class Models(BaseHTTPRequestHandler):
            def do_GET(self):
                body = json.dumps({"data": [{"id": "available-model"}]}).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *_args):
                pass

        server = HTTPServer(("127.0.0.1", 0), Models)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            endpoint = f"http://127.0.0.1:{server.server_port}"
            self.assertEqual(_test_configuration(endpoint, "available-model"), None)
            self.assertEqual(_test_configuration(endpoint, "missing-model"), "model_missing")
        finally:
            server.shutdown()
            thread.join()
            server.server_close()
        self.assertEqual(_test_configuration("http://127.0.0.1:1", "available-model"), "connection_failed")


class SyncConfigurationTests(unittest.TestCase):
    """Tests for sync_configuration: idempotency, ack, offline, stale, disabled handling."""

    def setUp(self):
        # Reset global state before each test
        import snitch_worker.transcription_config as tc
        tc._active = None
        # Patch test_configuration to succeed by default
        self._patch_test_config = patch("snitch_worker.transcription_config.test_configuration", return_value=None)
        self._patch_test_config.start()

    def tearDown(self):
        self._patch_test_config.stop()

    def _make_mock_client(self, config_response, status_response=None, status_status=200):
        """Create a mock AuthenticatedClient with configurable responses."""
        client = Mock()
        httpx_client = MagicMock()

        # Mock GET /api/worker/transcription-config
        get_response = Mock()
        get_response.status_code = config_response.get("status", 200)
        get_response.json.return_value = config_response.get("json", {})
        get_response.raise_for_status = Mock()
        if get_response.status_code >= 400:
            get_response.raise_for_status.side_effect = Exception("HTTP error")

        # Mock POST /api/worker/transcription-config/status
        post_response = Mock()
        post_response.status_code = status_status
        post_response.json.return_value = status_response or {}
        post_response.raise_for_status = Mock()
        if post_response.status_code >= 400:
            post_response.raise_for_status.side_effect = Exception("HTTP error")

        httpx_client.get.return_value = get_response
        httpx_client.post.return_value = post_response

        client.get_httpx_client.return_value = httpx_client
        return client, httpx_client, get_response, post_response

    def test_sync_configuration_idempotent_same_version_no_test_requested(self):
        """Calling sync_configuration twice with same version and no test_requested should not re-test."""
        config = {
            "status": 200,
            "json": {
                "mode": "enabled",
                "version": "v1",
                "endpoint": "http://whisper:8000",
                "model": "whisper-large-v3",
                "language": "fr",
                "test_requested": False,
            },
        }
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        # First call
        sync_configuration(client)
        # Second call - should not re-test or re-post
        sync_configuration(client)

        # GET called twice (once per call), POST called once (second call returns early)
        self.assertEqual(get_resp.raise_for_status.call_count, 2)
        self.assertEqual(post_resp.raise_for_status.call_count, 1)

        # Configuration should be active
        active = active_configuration()
        self.assertIsNotNone(active)
        self.assertEqual(active.version, "v1")

    def test_sync_configuration_re_tests_when_test_requested(self):
        """If test_requested is true, should re-test even with same version."""
        config = {
            "status": 200,
            "json": {
                "mode": "enabled",
                "version": "v1",
                "endpoint": "http://whisper:8000",
                "model": "whisper-large-v3",
                "language": "fr",
                "test_requested": True,
            },
        }
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        # Pre-activate with same version
        activate(ActiveConfiguration("v1", "http://whisper:8000", "whisper-large-v3", "fr"))

        sync_configuration(client)

        # Should still POST because test was requested
        self.assertEqual(post_resp.raise_for_status.call_count, 1)

    def test_sync_configuration_updates_on_new_version(self):
        """New version should trigger re-test and update."""
        config_v1 = {
            "status": 200,
            "json": {
                "mode": "enabled",
                "version": "v1",
                "endpoint": "http://whisper:8000",
                "model": "whisper-large-v3",
                "language": "fr",
                "test_requested": False,
            },
        }
        config_v2 = {
            "status": 200,
            "json": {
                "mode": "enabled",
                "version": "v2",
                "endpoint": "http://whisper:8000",
                "model": "whisper-large-v3-turbo",
                "language": "en",
                "test_requested": False,
            },
        }

        client, httpx_client, get_resp, post_resp = self._make_mock_client(config_v1)
        sync_configuration(client)

        # Update mock for v2
        get_resp.json.return_value = config_v2["json"]
        sync_configuration(client)

        # Should POST twice (once for each version)
        self.assertEqual(post_resp.raise_for_status.call_count, 2)

        active = active_configuration()
        self.assertEqual(active.version, "v2")
        self.assertEqual(active.model, "whisper-large-v3-turbo")

    def test_sync_configuration_returns_early_when_mode_not_enabled(self):
        """Mode disabled or unconfigured should return early without POST."""
        for mode in ["disabled", "unconfigured"]:
            with self.subTest(mode=mode):
                config = {
                    "status": 200,
                    "json": {
                        "mode": mode,
                        "version": "v1",
                        "endpoint": "http://whisper:8000",
                        "model": "whisper-large-v3",
                        "language": "fr",
                        "test_requested": False,
                    },
                }
                client, httpx_client, get_resp, post_resp = self._make_mock_client(config)
                sync_configuration(client)

                # Should not POST
                self.assertEqual(post_resp.raise_for_status.call_count, 0)

    def test_sync_configuration_handles_404_older_server(self):
        """404 response means server predates this contract - should return silently."""
        config = {"status": 404, "json": {}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)
        sync_configuration(client)

        # Should not POST
        self.assertEqual(post_resp.raise_for_status.call_count, 0)

    def test_sync_configuration_offline_server_raises(self):
        """Network error when fetching config should raise."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000", "model": "m", "language": "fr", "test_requested": False}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        # Make GET raise an exception
        get_resp.raise_for_status.side_effect = Exception("Connection refused")

        with self.assertRaises(Exception):
            sync_configuration(client)

        # Should not POST if GET fails
        self.assertEqual(post_resp.raise_for_status.call_count, 0)

    def test_sync_configuration_status_post_failure_raises(self):
        """Failure to POST status should raise and not activate."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000", "model": "whisper-large-v3", "language": "fr", "test_requested": False}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config, status_status=500)

        post_resp.raise_for_status.side_effect = Exception("Server error")

        with self.assertRaises(Exception):
            sync_configuration(client)

        # Should not activate on POST failure
        self.assertIsNone(active_configuration())

    def test_sync_configuration_activates_on_success(self):
        """Successful test should activate configuration."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000", "model": "whisper-large-v3", "language": "fr", "test_requested": False}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        sync_configuration(client)

        active = active_configuration()
        self.assertIsNotNone(active)
        self.assertEqual(active.version, "v1")
        self.assertEqual(active.endpoint, "http://whisper:8000")
        self.assertEqual(active.model, "whisper-large-v3")
        self.assertEqual(active.language, "fr")

    def test_sync_configuration_does_not_activate_on_test_failure(self):
        """Failed test (connection_failed or model_missing) should not activate."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000", "model": "whisper-large-v3", "language": "fr", "test_requested": False}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        # Patch test_configuration to return error
        with patch("snitch_worker.transcription_config.test_configuration", return_value="connection_failed"):
            sync_configuration(client)

        # Should POST with active=False
        post_call = httpx_client.post.call_args
        self.assertEqual(post_call.kwargs["json"]["active"], False)
        self.assertEqual(post_call.kwargs["json"]["error_code"], "connection_failed")

        # Should not activate
        self.assertIsNone(active_configuration())

    def test_sync_configuration_validates_response_fields(self):
        """Missing or invalid fields in response should raise ValueError."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000"}}  # missing model, language
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config)

        with self.assertRaises(ValueError):
            sync_configuration(client)

    def test_sync_configuration_stale_version_conflict(self):
        """If server version differs from what we POST, should get 409 and raise."""
        config = {"status": 200, "json": {"mode": "enabled", "version": "v1", "endpoint": "http://whisper:8000", "model": "whisper-large-v3", "language": "fr", "test_requested": False}}
        client, httpx_client, get_resp, post_resp = self._make_mock_client(config, status_status=409)

        post_resp.raise_for_status.side_effect = Exception("409 Conflict")

        with self.assertRaises(Exception):
            sync_configuration(client)

        # Should not activate on conflict
        self.assertIsNone(active_configuration())