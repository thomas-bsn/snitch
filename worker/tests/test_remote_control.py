"""Live journal and pause of a Worker (ADR 0063)."""
from __future__ import annotations

import json
import logging
import threading
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch

from snitch_client.models.error import Error

from snitch_worker import log, remote_control, runner


def record(message: str = "event", **fields) -> logging.LogRecord:
    entry = logging.LogRecord("snitch_worker", logging.INFO, __file__, 1, message, None, None)
    entry.fields = fields
    return entry


class UplinkTests(unittest.TestCase):
    def test_lines_emitted_before_the_client_exists_are_sent_in_order_from_zero(self):
        uplink = log.WorkerLogUplink(session_id="session-test-1")
        for i in range(3):
            uplink.emit(record(f"early_{i}"))
        sent = []
        uplink._send = lambda session, batch: sent.append((session, list(batch)))
        self.assertTrue(uplink._drain())
        self.assertEqual([seq for _, batch in sent for seq, _ in batch], [0, 1, 2])
        self.assertEqual(sent[0][0], "session-test-1")
        self.assertEqual(json.loads(sent[0][1][0][1])["event"], "early_0")

    def test_a_refused_batch_stays_and_is_sent_again_with_the_same_numbers(self):
        uplink = log.WorkerLogUplink(session_id="session-test-2")
        uplink.emit(record("a"))
        uplink.emit(record("b"))
        seen = []

        def refuse(session, batch):
            seen.append([seq for seq, _ in batch])
            raise RuntimeError("Snitch out of reach")
        uplink._send = refuse
        self.assertFalse(uplink._drain())
        uplink._send = lambda session, batch: seen.append([seq for seq, _ in batch])
        self.assertTrue(uplink._drain())
        self.assertEqual(seen, [[0, 1], [0, 1]])
        self.assertTrue(uplink._drain())  # nothing is left to send
        self.assertEqual(len(seen), 2)

    def test_batches_stay_within_what_the_server_accepts(self):
        uplink = log.WorkerLogUplink(session_id="session-test-3")
        for i in range(log.UPLINK_BATCH * 2 + 5):
            uplink.emit(record(f"line_{i}"))
        sizes = []
        uplink._send = lambda session, batch: sizes.append(len(batch))
        uplink._drain()
        self.assertEqual(sizes, [log.UPLINK_BATCH, log.UPLINK_BATCH, 5])

    def test_a_batch_is_cut_by_size_so_that_real_lines_stay_under_the_server_limit(self):
        uplink = log.WorkerLogUplink(session_id="session-test-9")
        for i in range(100):
            uplink.emit(record(f"line_{i}", stderr_tail="x" * 3000))
        sizes = []
        uplink._send = lambda session, batch: sizes.append(sum(len(line) for _, line in batch))
        uplink._drain()
        self.assertGreater(len(sizes), 1)
        self.assertTrue(all(size <= log.UPLINK_BATCH_BYTES + 4000 for size in sizes), sizes)

    def test_a_batch_snitch_refuses_for_good_is_dropped_and_does_not_block_the_next(self):
        uplink = log.WorkerLogUplink(session_id="session-test-10")
        for i in range(log.UPLINK_BATCH + 3):
            uplink.emit(record(f"line_{i}"))
        seen = []

        def refuse_first(session, batch):
            seen.append(batch[0][0])
            if batch[0][0] == 0:
                raise log.BatchRefused()
        uplink._send = refuse_first
        self.assertTrue(uplink._drain())
        self.assertEqual(seen, [0, log.UPLINK_BATCH])

    def test_the_status_of_the_answer_decides_between_waiting_and_dropping(self):
        def answer(status):
            return SimpleNamespace(status_code=status)
        for status, expected in ((204, None), (400, log.BatchRefused), (413, log.BatchRefused), (401, RuntimeError), (404, RuntimeError), (429, RuntimeError), (502, RuntimeError)):
            with patch.object(runner, "post_logs", return_value=answer(status)):
                if expected is None:
                    runner._push_logs(object(), "session-test-11", [(0, "{}")])
                else:
                    with self.assertRaises(expected):
                        runner._push_logs(object(), "session-test-11", [(0, "{}")])

    def test_the_buffer_is_bounded_and_drops_the_oldest(self):
        uplink = log.WorkerLogUplink(session_id="session-test-4")
        for i in range(log.UPLINK_BUFFER + 10):
            uplink.emit(record(f"line_{i}"))
        sent = []
        uplink._send = lambda session, batch: sent.extend(seq for seq, _ in batch)
        uplink._drain()
        self.assertEqual(len(sent), log.UPLINK_BUFFER)
        self.assertEqual(sent[0], 10)

    def test_a_long_line_is_cut_in_bytes_without_breaking_a_character(self):
        uplink = log.WorkerLogUplink(session_id="session-test-5")
        uplink.emit(record("é" * 5000))
        sent = []
        uplink._send = lambda session, batch: sent.extend(batch)
        uplink._drain()
        self.assertLessEqual(len(sent[0][1].encode("utf-8")), 4096)

    def test_secrets_never_leave_the_worker(self):
        uplink = log.WorkerLogUplink(session_id="session-test-6")
        uplink.emit(record("pip", stderr_tail="Bearer abcdef123456 https://user:pw@host/x /home/me/clip.wav", hf_token="hf_abcdefghijk"))
        sent = []
        uplink._send = lambda session, batch: sent.extend(batch)
        uplink._drain()
        text = sent[0][1]
        for secret in ("abcdef123456", "user:pw", "/home/me", "hf_abcdefghijk"):
            self.assertNotIn(secret, text)

    def test_the_thread_sends_and_close_flushes_the_end_of_the_session(self):
        uplink = log.WorkerLogUplink(session_id="session-test-7", interval=0.05)
        got: list[int] = []
        lock = threading.Lock()

        def take(session, batch):
            with lock:
                got.extend(seq for seq, _ in batch)
        uplink.start(take)
        uplink.emit(record("one"))
        deadline = time.monotonic() + 3
        while time.monotonic() < deadline and not got:
            time.sleep(0.02)
        self.assertEqual(got, [0])
        uplink.emit(record("two"))
        uplink.close()
        self.assertEqual(got, [0, 1])

    def test_a_failure_of_the_uplink_is_never_logged(self):
        logger = logging.getLogger("snitch_worker.uplink_test")
        logger.propagate = False
        uplink = log.WorkerLogUplink(session_id="session-test-8")
        logger.addHandler(uplink)
        uplink._send = lambda *_: (_ for _ in ()).throw(RuntimeError("down"))
        logger.warning("hello")
        uplink._drain()
        with uplink._lock:
            self.assertEqual(len(uplink._pending), 1)  # only the line itself: no line about the failure


def instruction(paused=False):
    return SimpleNamespace(paused=paused)


class ControlTests(unittest.TestCase):
    def control(self, wanted, acks, answer=None):
        control = remote_control.RemoteControl()
        return control, patch.multiple(remote_control, get_control=lambda client: wanted, ack_control=lambda client, body: (acks.append(body.paused), answer)[1])

    def test_a_pause_is_applied_and_confirmed_once(self):
        acks = []
        control, patches = self.control(instruction(paused=True), acks)
        with patches:
            control.sync_once(object())
            control.sync_once(object())
        self.assertTrue(control.paused.is_set())
        self.assertEqual(acks, [True])

    def test_a_resume_lifts_the_pause(self):
        acks = []
        control, patches = self.control(instruction(paused=False), acks)
        control.paused.set()
        with patches:
            control.sync_once(object())
        self.assertFalse(control.paused.is_set())
        self.assertEqual(acks, [False])  # the first poll of a process tells Snitch it runs

    def test_a_change_is_confirmed_again(self):
        acks, wanted = [], [instruction(paused=True)]
        control = remote_control.RemoteControl()
        with patch.multiple(remote_control, get_control=lambda client: wanted[0], ack_control=lambda client, body: acks.append(body.paused)):
            control.sync_once(object())
            wanted[0] = instruction(paused=False)
            control.sync_once(object())
        self.assertEqual(acks, [True, False])

    def test_a_refused_acknowledgement_is_tried_again_at_the_next_poll(self):
        acks = []
        control, patches = self.control(instruction(paused=True), acks, answer=Error(error="worker_token_invalid"))
        with patches:
            with self.assertRaises(RuntimeError):
                control.sync_once(object())
            with self.assertRaises(RuntimeError):
                control.sync_once(object())
        self.assertEqual(acks, [True, True])

    def test_an_unreadable_answer_is_an_error_not_a_state(self):
        control = remote_control.RemoteControl()
        with patch.multiple(remote_control, get_control=lambda client: Error(error="worker_token_invalid")):
            with self.assertRaises(RuntimeError):
                control.sync_once(object())

    def test_the_loop_outlives_errors_and_logs_the_outage_once(self):
        calls = {"n": 0}
        stop = threading.Event()

        class Flaky:
            def sync_once(self, client):
                calls["n"] += 1
                if calls["n"] >= 4:
                    stop.set()
                if calls["n"] < 3:
                    raise OSError("down")
        with self.assertLogs("snitch_worker", level="INFO") as captured:
            remote_control.control_loop(object(), Flaky(), stop, 0.01)
        events = [json.loads(log.JsonFormatter().format(r))["event"] for r in captured.records]
        self.assertEqual(events.count("worker_control_unreachable"), 1)
        self.assertEqual(events.count("worker_control_reconnected"), 1)

    def test_a_paused_worker_claims_nothing(self):
        control = remote_control.RemoteControl()
        control.paused.set()
        stop = threading.Event()
        claimed = []
        with patch.object(runner, "run_once", side_effect=lambda *a, **k: claimed.append(1) or True):
            thread = threading.Thread(target=runner._worker_loop, args=(object(), 1, stop, 0.01, 1.0, None, None, control))
            thread.start()
            time.sleep(0.15)
            self.assertEqual(claimed, [])
            control.paused.clear()  # resumed
            deadline = time.monotonic() + 3
            while time.monotonic() < deadline and not claimed:
                time.sleep(0.01)
            stop.set()
            thread.join(timeout=3)
        self.assertTrue(claimed)


if __name__ == "__main__":
    unittest.main()
