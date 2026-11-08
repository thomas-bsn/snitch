"""Concurrent reception and completion preserve both durable outcomes."""
import json
import threading
import unittest

from test_model_orders import Harness, FakeSnitch, ORDER, order
from snitch_worker import model_orders, voice_models


class OrderConcurrencyTests(unittest.TestCase):
    def test_reception_cannot_overwrite_a_parallel_completion(self):
        downloading, finish_download, acknowledging, finish_ack, reported = [threading.Event() for _ in range(5)]
        def download(repo, revision, target, progress):
            if revision == ORDER["revision"]:
                downloading.set()
                if not finish_download.wait(5):
                    raise TimeoutError()
        h = Harness(self, download=download)
        second = order(order_id="ord_synthetic_order_identifier_0002", idempotency_key="install-c-0002", repo="synthetic/other", revision="c" * 40)
        class Snitch(FakeSnitch):
            def post(self, path, json=None, **kwargs):
                if second["order_id"] in path and path.endswith("/ack"):
                    acknowledging.set()
                    if not finish_ack.wait(5):
                        raise TimeoutError()
                if path.endswith("/status") and json.get("state") == "ready" and json.get("revision") == ORDER["revision"]:
                    reported.set()
                return super().post(path, json=json, **kwargs)
        snitch = Snitch([ORDER])
        model_orders.sync_orders(snitch, h.executor)
        self.assertTrue(downloading.wait(5))
        snitch.orders = [second]
        receiver = threading.Thread(target=lambda: model_orders.sync_orders(snitch, h.executor))
        receiver.start()
        try:
            self.assertTrue(acknowledging.wait(5))
            finish_download.set()
            self.assertTrue(reported.wait(5))
        finally:
            finish_ack.set()
            finish_download.set()
            receiver.join(5)
            h.executor.wait()
        with open(model_orders.state_path(h.env)) as f:
            state = json.load(f)
        self.assertEqual(set(state["orders"]), {ORDER["order_id"], second["order_id"]})
        self.assertEqual(len(state["keys"]), 2)
        restarted = Harness(self, directory=h.directory)
        restarted.sync(Snitch([ORDER, second]))
        self.assertEqual(restarted.downloads, [])

    def test_validated_model_needs_no_download_space(self):
        h = Harness(self)
        h.sync(FakeSnitch([ORDER]))
        h.executor._free_bytes = lambda _path: 0
        snitch = FakeSnitch([order(order_id="ord_synthetic_order_identifier_0002", idempotency_key="recheck-ready-0002", size_bytes=100_000_000)])
        h.sync(snitch)
        self.assertEqual(snitch.states(), ["ready"])
        self.assertEqual(len(h.downloads), 1)
