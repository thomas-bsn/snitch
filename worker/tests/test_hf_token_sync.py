"""Central Hugging Face token on the Worker (#194, ADR 0027). Synthetic values only."""

import json
import logging
import os
import stat
import tempfile
import unittest
from unittest.mock import patch

import httpx

from snitch_worker import hf_token_sync, voice_download

SYNTHETIC = "hf_SyntheticTokenForTestsOnly0123456789abcdef"
REPLACEMENT = "hf_SyntheticReplacementTokenOnly0123456789"


class FakeResponse:
    def __init__(self, status_code=200, payload=None):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload

    def raise_for_status(self):
        if self.status_code >= 400:
            request = httpx.Request("GET", "http://snitch.test/")
            raise httpx.HTTPStatusError("refused", request=request, response=httpx.Response(self.status_code, request=request))


class FakeHTTP:
    """Answers the three central-token routes; records every call so tests can assert what was asked."""

    def __init__(self, version="v1", token=SYNTHETIC, status=200):
        self.version = version
        self.token = token
        self.status = status
        self.calls = []
        self.acks = []

    def get(self, path, **_kwargs):
        self.calls.append(("GET", path))
        if self.status != 200:
            return FakeResponse(self.status)
        if path == "/api/worker/hf-token/version":
            return FakeResponse(200, {"version": self.version})
        if path == "/api/worker/hf-token":
            return FakeResponse(200, {"version": self.version, "token": self.token})
        return FakeResponse(404)

    def post(self, path, json=None, **_kwargs):
        self.calls.append(("POST", path))
        if path == "/api/worker/hf-token/ack":
            self.acks.append(json)
            return FakeResponse(204)
        return FakeResponse(404)


class FakeClient:
    def __init__(self, http):
        self._http = http

    def get_httpx_client(self):
        return self._http


class HFTokenSyncTests(unittest.TestCase):
    def setUp(self):
        self._dir = tempfile.TemporaryDirectory()
        self.secrets = os.path.join(self._dir.name, ".secrets")
        self.token_file = os.path.join(self.secrets, "hf_token")
        self.logged = []
        patcher = patch.object(hf_token_sync.log, "emit", side_effect=lambda level, event, **fields: self.logged.append((event, fields)))
        patcher.start()
        self.addCleanup(patcher.stop)

    def tearDown(self):
        self._dir.cleanup()

    def read(self, path):
        with open(path, encoding="utf-8") as handle:
            return handle.read()

    def assert_no_value_anywhere(self, http=None, values=(SYNTHETIC, REPLACEMENT)):
        text = json.dumps(self.logged, default=str)
        for value in values:
            self.assertNotIn(value, text)
            if http is not None:
                self.assertNotIn(value, json.dumps(http.acks))
                self.assertNotIn(value, json.dumps(http.calls))

    # Negative cases first: nothing is written and nothing is fetched without the server's answer.

    def test_unauthenticated_or_revoked_worker_writes_nothing(self):
        for status in (401, 403):
            http = FakeHTTP(status=status)
            with self.assertRaises(httpx.HTTPStatusError):
                hf_token_sync.sync(FakeClient(http), self.token_file)
            self.assertFalse(os.path.exists(self.token_file))
            self.assertNotIn(("GET", "/api/worker/hf-token"), http.calls)
            self.assertEqual(http.acks, [])
        self.assert_no_value_anywhere()

    def test_unreadable_or_absent_central_token_purges_nothing(self):
        self.secrets_setup(central_version="v1")
        http = FakeHTTP(version="0")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertTrue(os.path.isfile(self.token_file))
        self.assertNotIn(("GET", "/api/worker/hf-token"), http.calls)
        self.assertEqual(http.acks, [])

    def test_local_token_is_never_replaced_silently(self):
        self.write_local()
        http = FakeHTTP(version="v7", token=SYNTHETIC)
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(self.read(self.token_file), "hf_LocalOnlyTokenForTests0123456789")
        self.assertEqual(http.acks[-1]["outcome"], "kept_local")
        self.assertEqual(http.acks[-1]["source"], "local")
        self.assertEqual(http.acks[-1]["version"], "v7")
        self.assert_no_value_anywhere(http)

    def test_local_marker_protects_the_token_from_a_later_purge(self):
        self.write_central("v1", SYNTHETIC)
        hf_token_sync.mark_local(self.token_file)
        self.write_local()
        http = FakeHTTP(version="v2", token=None)
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(self.read(self.token_file), "hf_LocalOnlyTokenForTests0123456789")
        self.assertEqual(http.acks[-1]["outcome"], "kept_local")

    def test_failed_store_is_acknowledged_without_the_value(self):
        http = FakeHTTP(version="v3")
        with patch.object(voice_download, "persist_token", return_value=False):
            hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertFalse(os.path.exists(self.token_file))
        self.assertEqual(http.acks[-1]["outcome"], "failed")
        self.assertEqual(http.acks[-1]["error_code"], "hf_token_store_failed")
        self.assert_no_value_anywhere(http)

    def test_malformed_value_from_server_is_refused(self):
        http = FakeHTTP(version="v4", token="hf_bad value with spaces")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertFalse(os.path.exists(self.token_file))
        self.assertEqual(http.acks[-1]["outcome"], "failed")
        self.assertNotIn("bad value", json.dumps(self.logged + http.acks))

    # Positive cases.

    def test_new_worker_obtains_the_token_and_acknowledges_it(self):
        http = FakeHTTP(version="v1")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(self.read(self.token_file), SYNTHETIC)
        self.assertEqual(stat.S_IMODE(os.stat(self.token_file).st_mode), 0o600)
        self.assertEqual(stat.S_IMODE(os.stat(self.secrets).st_mode), 0o700)
        self.assertEqual(http.acks, [{"version": "v1", "outcome": "applied", "source": "central", "error_code": ""}])
        self.assert_no_value_anywhere(http)

    def test_acknowledged_version_is_not_fetched_again(self):
        http = FakeHTTP(version="v1")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        http.calls.clear()
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(http.calls, [("GET", "/api/worker/hf-token/version")])

    def test_central_replacement_overwrites_only_a_central_copy(self):
        self.write_central("v1", SYNTHETIC)
        http = FakeHTTP(version="v2", token=REPLACEMENT)
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(self.read(self.token_file), REPLACEMENT)
        self.assertEqual(http.acks[-1]["outcome"], "applied")
        self.assert_no_value_anywhere(http)

    def test_central_removal_purges_the_central_copy_only(self):
        self.write_central("v1", SYNTHETIC)
        http = FakeHTTP(version="v2", token=None)
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertFalse(os.path.exists(self.token_file))
        self.assertEqual(http.acks[-1]["outcome"], "removed")
        self.assertEqual(http.acks[-1]["source"], "none")
        # Acknowledged: the removal is not fetched or acknowledged again.
        http.calls.clear()
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertEqual(http.calls, [("GET", "/api/worker/hf-token/version")])

    def test_version_moving_during_the_exchange_is_ignored(self):
        class Moving(FakeHTTP):
            def get(self, path, **kwargs):
                if path == "/api/worker/hf-token":
                    return FakeResponse(200, {"version": "v9", "token": SYNTHETIC})
                return super().get(path, **kwargs)

        http = Moving(version="v8")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertFalse(os.path.exists(self.token_file))
        self.assertEqual(http.acks, [])

    def test_old_server_without_the_endpoint_is_tolerated(self):
        http = FakeHTTP(status=404)
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertFalse(os.path.exists(self.token_file))

    def test_value_is_not_exported_to_the_environment_or_arguments(self):
        http = FakeHTTP(version="v1")
        hf_token_sync.sync(FakeClient(http), self.token_file)
        self.assertNotIn(SYNTHETIC, json.dumps(dict(os.environ)))
        env = voice_download.without_token(dict(os.environ), keep_file=True)
        self.assertNotIn(SYNTHETIC, json.dumps(env))

    # Helpers.

    def secrets_setup(self, central_version):
        self.write_central(central_version, SYNTHETIC)

    def write_central(self, version, token):
        os.makedirs(self.secrets, mode=0o700, exist_ok=True)
        if token:
            with open(self.token_file, "w", encoding="utf-8") as handle:
                handle.write(token)
        hf_token_sync.write_meta(self.token_file, "central", version, acked=True)

    def write_local(self):
        os.makedirs(self.secrets, mode=0o700, exist_ok=True)
        with open(self.token_file, "w", encoding="utf-8") as handle:
            handle.write("hf_LocalOnlyTokenForTests0123456789")


if __name__ == "__main__":
    logging.disable(logging.CRITICAL)
    unittest.main()
