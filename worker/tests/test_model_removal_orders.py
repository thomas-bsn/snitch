"""The Worker removes a model only on a remove order from Snitch, and checks its disk before an install (ADR 0038).

Synthetic values only; no network."""

import os
import unittest
from unittest.mock import Mock

from snitch_worker import voice_models

from test_model_orders import ORDER, REPO, REVISION, FakeSnitch, Harness, order

REMOVE = order(
    order_id="ord_synthetic_remove_identifier_0001",
    action="remove",
    idempotency_key="remove-a-0001",
    status="pending",
)


def installed(h: Harness) -> str:
    h.sync(FakeSnitch([ORDER]))
    path = voice_models.model_path(h.env, REPO, REVISION)
    assert os.path.isdir(path)
    return path


class RemovalOrderTests(unittest.TestCase):
    def test_a_model_is_removed_only_on_a_remove_order_and_reported_with_it(self):
        h = Harness(self)
        path = installed(h)
        snitch = FakeSnitch([REMOVE])
        h.sync(snitch)
        self.assertFalse(os.path.exists(path))
        self.assertEqual(snitch.acks, [(REMOVE["order_id"], {"accepted": True, "cause_code": ""})])
        self.assertEqual([r["state"] for r in snitch.reports], ["removed"])
        self.assertEqual(snitch.reports[0]["order_id"], REMOVE["order_id"])
        self.assertEqual(snitch.reports[0]["model_id"], "test-b")

    def test_the_pull_alone_removes_nothing(self):
        h = Harness(self)
        path = installed(h)
        client = Mock()
        http = client.get_httpx_client.return_value
        http.get.return_value = Mock(status_code=200, json=Mock(return_value={"models": [], "remove": [{"model_id": "test-b", "repo": REPO, "revision": REVISION}], "orders": []}))
        voice_models.sync_voice_models(client, h.manager)
        self.assertTrue(os.path.isdir(path))
        http.post.assert_not_called()

    def test_a_replayed_remove_order_is_not_run_again_after_a_restart(self):
        h = Harness(self)
        installed(h)
        h.sync(FakeSnitch([REMOVE]))
        restarted = Harness(self, directory=h.directory)
        restarted.sync(FakeSnitch([ORDER]))  # the install is also remembered, so nothing comes back
        snitch = FakeSnitch([REMOVE])
        restarted.sync(snitch)
        self.assertEqual([r["state"] for r in snitch.reports], ["removed"])
        self.assertEqual(restarted.downloads, [])

    def test_a_removal_that_leaves_the_files_is_reported_as_failed(self):
        h = Harness(self)
        path = installed(h)
        original = voice_models.shutil.rmtree
        voice_models.shutil.rmtree = lambda *_a, **_k: None  # the volume refuses the deletion
        self.addCleanup(setattr, voice_models.shutil, "rmtree", original)
        snitch = FakeSnitch([REMOVE])
        h.sync(snitch)
        self.assertTrue(os.path.isdir(path))
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "models_volume_unwritable"))

    def test_a_remove_order_aimed_at_another_worker_or_malformed_is_refused(self):
        h = Harness(self)
        path = installed(h)
        snitch = FakeSnitch([{**REMOVE, "worker_id": "wk_someone_else_identifier_000001"}, {**REMOVE, "order_id": "ord_synthetic_remove_identifier_0002", "idempotency_key": "remove-a-0002", "revision": "main"}])
        h.sync(snitch)
        self.assertTrue(os.path.isdir(path))
        self.assertEqual([accepted["accepted"] for _id, accepted in snitch.acks], [False, False])
        self.assertEqual(snitch.reports, [])


class DiskCheckTests(unittest.TestCase):
    def test_an_install_that_does_not_fit_fails_with_disk_full_and_downloads_nothing(self):
        h = Harness(self)
        h.executor._free_bytes = lambda _path: 10_000_000
        snitch = FakeSnitch([{**ORDER, "size_bytes": 30_000_000}])
        h.sync(snitch)
        self.assertEqual(h.downloads, [])
        self.assertEqual((snitch.reports[-1]["state"], snitch.reports[-1]["cause_code"]), ("failed", "disk_full"))

    def test_an_install_that_fits_runs(self):
        h = Harness(self)
        h.executor._free_bytes = lambda _path: 10_000_000_000
        snitch = FakeSnitch([{**ORDER, "size_bytes": 30_000_000}])
        h.sync(snitch)
        self.assertEqual(snitch.states()[-1], "ready")


if __name__ == "__main__":
    unittest.main()
