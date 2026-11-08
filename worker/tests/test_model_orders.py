"""The Worker runs the install orders Snitch gives it and nothing else (ADR 0038). Synthetic values only."""

import copy
import io
import json
import logging
import os
import stat
import tempfile
import unittest

from snitch_worker import log, model_orders, voice_download, voice_models

WORKER = "wk_synthetic_worker_identifier_0001"
REPO = "synthetic/diarization-b"
REVISION = "b" * 40
SECRET = "hf_SyntheticOrderTokenOnly0123456789abcdef"
ORDER = {
    "order_id": "ord_synthetic_order_identifier_0001",
    "worker_id": WORKER,
    "kind": "diarization",
    "action": "install",
    "model_id": "test-b",
    "engine": "pyannote",
    "repo": REPO,
    "revision": REVISION,
    "idempotency_key": "install-b-0001",
    "status": "pending",
}


class FakeResponse:
    def __init__(self, status_code=200, payload=None):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload

    def raise_for_status(self):
        if self.status_code >= 400:
            raise RuntimeError("refused")


class FakeSnitch:
    """A Snitch that gives out orders and records every acknowledgement and report."""

    def __init__(self, orders=(), worker_id=WORKER):
        self.orders = [copy.deepcopy(o) for o in orders]
        self.worker_id = worker_id
        self.acks: list[tuple[str, dict]] = []
        self.reports: list[dict] = []

    def get_httpx_client(self):
        return self

    def get(self, path, **_kwargs):
        assert path == "/api/worker/voice-models"
        return FakeResponse(200, {"worker_id": self.worker_id, "models": [], "remove": [], "orders": copy.deepcopy(self.orders)})

    def post(self, path, json=None, **_kwargs):
        if path.startswith("/api/worker/voice-model-orders/") and path.endswith("/ack"):
            self.acks.append((path.split("/")[-2], json))
        elif path == "/api/worker/voice-models/status":
            self.reports.append(json)
        else:
            return FakeResponse(404)
        return FakeResponse(204)

    def states(self):
        return [r["state"] for r in self.reports]


class Harness:
    def __init__(self, test: unittest.TestCase, download=None, verify=None, directory=None):
        self.directory = directory or tempfile.TemporaryDirectory()
        if directory is None:
            test.addCleanup(self.directory.cleanup)
        self.env = {"SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(self.directory.name, "embedding")}
        self.token_file = os.path.join(self.directory.name, ".secrets", "hf_token")
        self.downloads: list[tuple[str, str, str]] = []
        self.verified: list[tuple[str, dict]] = []

        def default_download(repo, revision, target, progress):
            self.downloads.append((repo, revision, target))
            progress(10_000_000, 30_000_000)
            os.makedirs(target, exist_ok=True)
            with open(os.path.join(target, "weights.bin"), "w") as handle:
                handle.write("synthetic")
            progress(30_000_000, 30_000_000)

        def default_verify(path, environ):
            self.verified.append((path, dict(environ)))

        self.manager = voice_models.ModelManager(self.env, download or default_download, verify or default_verify, None, min_progress_interval=0)
        self.executor = model_orders.OrderExecutor(self.manager, self.env, self.token_file)

    def sync(self, snitch: FakeSnitch):
        model_orders.sync_orders(snitch, self.executor)
        self.executor.wait()


def order(**changes):
    value = copy.deepcopy(ORDER)
    value.update(changes)
    return value


class ExecutionTests(unittest.TestCase):
    def test_nothing_is_downloaded_without_an_order(self):
        h = Harness(self)
        snitch = FakeSnitch()
        h.sync(snitch)
        self.assertEqual((h.downloads, h.verified, snitch.reports, snitch.acks), ([], [], [], []))

    def test_an_order_is_acknowledged_downloaded_verified_then_ready(self):
        h = Harness(self)
        snitch = FakeSnitch([ORDER])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [(ORDER["order_id"], {"accepted": True, "cause_code": ""})])
        self.assertEqual(snitch.states()[0], "downloading")
        self.assertEqual(snitch.states()[-2:], ["verifying", "ready"])
        self.assertTrue(all(r["order_id"] == ORDER["order_id"] and r["model_id"] == "test-b" and r["revision"] == REVISION for r in snitch.reports))
        self.assertGreater(snitch.reports[-1]["occupied_bytes"], 0)
        self.assertEqual(len(h.downloads), 1)
        # ready only after the synthetic inference ran on what was downloaded
        self.assertEqual(h.verified[0][0], h.downloads[0][2])

    def test_ready_is_never_reported_when_the_inference_fails(self):
        class Reason(Exception):
            reason = "inference_failed"

        def verify(path, environ):
            raise Reason()

        h = Harness(self, verify=verify)
        snitch = FakeSnitch([ORDER])
        h.sync(snitch)
        self.assertNotIn("ready", snitch.states())
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "inference_failed"))

    def test_a_download_failure_carries_its_cause(self):
        def download(repo, revision, target, progress):
            raise voice_models.DownloadFailed("hf_auth_required")

        h = Harness(self, download=download)
        snitch = FakeSnitch([ORDER])
        h.sync(snitch)
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "hf_auth_required"))
        self.assertEqual(h.verified, [])

    def test_a_replayed_order_is_not_executed_twice_even_after_a_restart(self):
        h = Harness(self)
        h.sync(FakeSnitch([ORDER]))
        h.sync(FakeSnitch([ORDER]))
        self.assertEqual(len(h.downloads), 1)
        restarted = Harness(self, directory=h.directory)
        snitch = FakeSnitch([ORDER])
        restarted.sync(snitch)
        self.assertEqual(restarted.downloads, [])
        self.assertEqual(restarted.verified, [])
        # the outcome Snitch may have lost is told again, never recomputed
        self.assertEqual(snitch.states(), ["ready"])

    def test_a_failure_survives_a_restart_without_another_download(self):
        calls = []

        def failing(repo, revision, target, progress):
            calls.append(target)
            raise voice_models.DownloadFailed("hf_network")

        h = Harness(self, download=failing)
        h.sync(FakeSnitch([ORDER]))
        restarted = Harness(self, download=failing, directory=h.directory)
        snitch = FakeSnitch([ORDER])
        restarted.sync(snitch)
        self.assertEqual(len(calls), 1)
        self.assertEqual(snitch.states(), ["failed"])
        self.assertEqual(snitch.reports[0]["cause_code"], "hf_network")

    def test_an_order_cut_short_by_a_restart_is_run_once_more_without_a_second_acknowledgement(self):
        h = Harness(self)
        # Snitch holds the order as acknowledged: the earlier run never wrote an outcome.
        resumed = order(status="acknowledged")
        snitch = FakeSnitch([resumed])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [])
        self.assertEqual(snitch.states()[-1], "ready")
        self.assertEqual(len(h.downloads), 1)

    def test_a_new_order_after_a_failure_runs_again(self):
        attempts = []

        def download(repo, revision, target, progress):
            attempts.append(target)
            if len(attempts) == 1:
                raise voice_models.DownloadFailed("hf_network")
            os.makedirs(target, exist_ok=True)

        h = Harness(self, download=download)
        h.sync(FakeSnitch([ORDER]))
        snitch = FakeSnitch([order(order_id="ord_synthetic_order_identifier_0002", idempotency_key="retry-b-0002")])
        h.sync(snitch)
        self.assertEqual(len(attempts), 2)
        self.assertEqual(snitch.states()[-1], "ready")


class ReceptionChecksTests(unittest.TestCase):
    def refused(self, bad, **harness):
        h = Harness(self, **harness)
        snitch = FakeSnitch([bad])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [(bad.get("order_id", ""), {"accepted": False, "cause_code": "invalid_order"})] if bad.get("order_id", "").startswith("ord_") else snitch.acks)
        self.assertEqual((h.downloads, h.verified), ([], []))
        self.assertNotIn("downloading", snitch.states())
        return snitch

    def test_an_order_for_another_worker_is_refused(self):
        snitch = self.refused(order(worker_id="wk_another_worker_identifier_0002"))
        self.assertEqual(len(snitch.acks), 1)

    def test_identity_type_model_and_revision_are_checked(self):
        for bad in (
            order(action="purge"),
            order(kind="separation", engine="pyannote"),
            order(kind="fingerprint", engine="nemo"),
            order(kind="unknown"),
            order(engine="free-form"),
            order(repo="not a repository"),
            order(repo="../../etc/passwd"),
            order(revision="main"),
            order(revision="B" * 40),
            order(model_id=""),
            order(idempotency_key=""),
        ):
            with self.subTest(bad=bad):
                snitch = self.refused(bad)
                self.assertEqual(len(snitch.acks), 1)

    def test_an_order_without_a_valid_identity_is_not_acknowledged_but_still_not_run(self):
        snitch = self.refused(order(order_id="forged"))
        self.assertEqual(snitch.acks, [])

    def test_the_same_key_for_another_content_is_refused(self):
        h = Harness(self)
        h.sync(FakeSnitch([ORDER]))
        other = order(order_id="ord_synthetic_order_identifier_0003", model_id="test-c", repo="synthetic/other", revision="c" * 40)
        snitch = FakeSnitch([other])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [(other["order_id"], {"accepted": False, "cause_code": "invalid_order"})])
        self.assertEqual(len(h.downloads), 1)

    def test_the_same_identity_cannot_change_its_action(self):
        h = Harness(self)
        h.sync(FakeSnitch([ORDER]))
        snitch = FakeSnitch([order(action="remove")])
        h.sync(snitch)
        self.assertEqual(snitch.acks, [(ORDER["order_id"], {"accepted": False, "cause_code": "invalid_order"})])
        self.assertEqual(len(h.downloads), 1)

    def test_the_volume_is_pinned_to_the_first_identity_it_sees(self):
        h = Harness(self)
        h.sync(FakeSnitch([ORDER]))
        intruder = order(order_id="ord_synthetic_order_identifier_0004", worker_id="wk_another_worker_identifier_0002", idempotency_key="install-z-0004")
        snitch = FakeSnitch([intruder], worker_id="wk_another_worker_identifier_0002")
        h.sync(snitch)
        self.assertEqual(snitch.acks, [])
        self.assertEqual(len(h.downloads), 1)


class TokenTests(unittest.TestCase):
    def test_the_token_of_an_order_is_kept_in_the_secret_file_and_nowhere_else(self):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        h = Harness(self)
        snitch = FakeSnitch([order(hf_token=SECRET)])
        h.sync(snitch)
        with open(h.token_file, encoding="utf-8") as handle:
            self.assertEqual(handle.read().strip(), SECRET)
        self.assertEqual(stat.S_IMODE(os.stat(h.token_file).st_mode), 0o600)
        self.assertNotIn(SECRET, stream.getvalue())
        self.assertNotIn(SECRET, json.dumps(snitch.reports) + json.dumps(snitch.acks))
        self.assertNotIn(SECRET, json.dumps(h.downloads))
        for _path, env in h.verified:
            self.assertNotIn(SECRET, json.dumps(env))
            self.assertNotIn(voice_download.TOKEN_FILE_ENV, env)
        self.assertNotIn(SECRET, json.dumps(dict(os.environ)))

    def test_a_redelivered_acknowledged_order_restores_a_token_lost_in_a_crash(self):
        h = Harness(self)
        h.sync(FakeSnitch([order(hf_token=SECRET, status="acknowledged")]))
        with open(h.token_file, encoding="utf-8") as handle:
            self.assertEqual(handle.read().strip(), SECRET)

    def test_a_local_token_is_never_replaced_by_an_order(self):
        h = Harness(self)
        voice_download.persist_token(h.token_file, "hf_LocalTokenOnly0123456789")
        from snitch_worker import hf_token_sync

        hf_token_sync.mark_local(h.token_file)
        h.sync(FakeSnitch([order(hf_token=SECRET)]))
        with open(h.token_file, encoding="utf-8") as handle:
            self.assertEqual(handle.read().strip(), "hf_LocalTokenOnly0123456789")

    def test_an_invalid_token_is_ignored_without_being_logged(self):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        self.addCleanup(log.configure, "INFO", io.StringIO())
        h = Harness(self)
        h.sync(FakeSnitch([order(hf_token="not a token with spaces")]))
        self.assertFalse(os.path.exists(h.token_file))
        self.assertNotIn("not a token", stream.getvalue())


if __name__ == "__main__":
    unittest.main()
