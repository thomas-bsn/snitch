import io
import threading
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch

import httpx
from snitch_client.models.error import Error

from snitch_worker import log, runner


class PresenceAndLeaseTests(unittest.TestCase):
    """ADR 0029: presence is independent of jobs; a lost lease cancels without a result."""

    LEASE = "lt_synthetic-lease-token"

    def setUp(self):
        self.addCleanup(log.configure, "INFO", io.StringIO())
        log.configure("INFO", io.StringIO())
        self.client = SimpleNamespace(_base_url="https://snitch.example")
        self.task = SimpleNamespace(
            id=7, clip_id=3, step="fixture.echo", input_version="v1", cycle=1, attempts=1, lease_owner="wk_synthetic"
        )
        self.claim = SimpleNamespace(job=self.task, lease_token=self.LEASE)

    def test_presence_retry_delay_grows_and_is_capped_at_ten_seconds(self):
        delays = [runner.presence_retry_delay(attempt) for attempt in range(1, 8)]
        self.assertEqual(delays, [1, 2, 4, 8, 10, 10, 10])

    def test_presence_pulses_on_its_own_thread_while_a_job_runs(self):
        stop = threading.Event()
        release = threading.Event()
        pulses = []

        def long_job(*_args, **_kwargs):
            release.wait(2)
            return True

        def record_pulse(**_kwargs):
            pulses.append(time.monotonic())
            if len(pulses) >= 4:
                release.set()
                stop.set()
            return object()

        with (
            patch.object(runner, "pulse", side_effect=record_pulse),
            patch.object(runner, "run_once", side_effect=long_job),
        ):
            runner.run_until_stopped(self.client, 1, stop, poll_interval=0.01, presence_interval=0.01)
        self.assertGreaterEqual(len(pulses), 4)

    def test_presence_keeps_pulsing_after_network_errors(self):
        stop = threading.Event()
        calls = []

        def flaky_pulse(**_kwargs):
            calls.append(1)
            if len(calls) <= 2:
                raise httpx.ConnectError("synthetic outage")
            if len(calls) >= 4:
                stop.set()
            return object()

        with (
            patch.object(runner, "pulse", side_effect=flaky_pulse),
            patch.object(runner, "run_once", return_value=False),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            runner.run_until_stopped(self.client, 1, stop, poll_interval=0.01, presence_interval=0.01)
        self.assertGreaterEqual(len(calls), 4)

    def test_network_error_on_lease_renewal_is_retried_and_job_completes(self):
        outcomes = [httpx.ConnectError("synthetic outage")]

        def heartbeat(**_kwargs):
            value = outcomes.pop(0) if outcomes else object()
            if isinstance(value, Exception):
                raise value
            return value

        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", side_effect=heartbeat) as heartbeat_mock,
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.2) or {"provider": "fixture"}),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        self.assertGreaterEqual(heartbeat_mock.call_count, 2)
        complete.assert_called_once()
        fail.assert_not_called()

    def test_refused_lease_cancels_the_job_without_any_result(self):
        refusal = Error.from_dict({"error": "lease_not_owned"})
        started = time.monotonic()
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", return_value=refusal),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.3) or {"provider": "fixture"}),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        self.assertLess(time.monotonic() - started, 10)
        complete.assert_not_called()
        fail.assert_not_called()

    def test_result_is_discarded_when_the_lease_expired_during_an_outage(self):
        with (
            patch.object(runner, "pulse", return_value=object()),
            patch.object(runner, "claim_job", return_value=self.claim),
            patch.object(runner, "get_job_input", return_value=SimpleNamespace(to_dict=lambda: {})),
            patch.object(runner, "heartbeat_job", side_effect=httpx.ConnectError("synthetic outage")),
            patch.object(runner, "complete_job", return_value=object()) as complete,
            patch.object(runner, "fail_job", return_value=object()) as fail,
            patch.object(runner, "fake_provider", side_effect=lambda *_: time.sleep(0.3) or {"provider": "fixture"}),
            patch.object(runner, "LEASE_TTL", 0.05),
            patch.object(runner, "PRESENCE_RETRY_CAP", 0.01),
        ):
            self.assertTrue(runner.run_once(self.client, heartbeat_interval=0.01))

        complete.assert_not_called()
        fail.assert_not_called()
