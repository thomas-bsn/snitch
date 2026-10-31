import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer

from snitch_worker.transcription_config import test_configuration


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
            self.assertEqual(test_configuration(endpoint, "available-model"), None)
            self.assertEqual(test_configuration(endpoint, "missing-model"), "model_missing")
        finally:
            server.shutdown()
            thread.join()
            server.server_close()
        self.assertEqual(test_configuration("http://127.0.0.1:1", "available-model"), "connection_failed")
