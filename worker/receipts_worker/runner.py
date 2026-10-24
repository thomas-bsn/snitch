"""Small HTTPS polling loop for the phase 2 fake provider."""

from __future__ import annotations

import os
import signal
import ssl
import threading
from concurrent.futures import ThreadPoolExecutor
from typing import Any
from urllib.parse import urlparse

import httpx
from receipts_api.api.default.claim_worker_job import sync as claim_job
from receipts_api.api.default.complete_worker_job import sync as complete_job
from receipts_api.api.default.fail_worker_job import sync as fail_job
from receipts_api.api.default.get_worker_job_input import sync as get_job_input
from receipts_api.api.default.heartbeat_worker_job import sync as heartbeat_job
from receipts_api.api.default.worker_pulse import sync as pulse
from receipts_api.client import AuthenticatedClient
from receipts_api.models.error import Error
from receipts_api.models.worker_failure_input import WorkerFailureInput
from receipts_api.models.worker_pulse_input import WorkerPulseInput
from receipts_api.models.worker_result_input import WorkerResultInput
from receipts_api.models.worker_result_input_result import WorkerResultInputResult


def fake_provider(step: str) -> dict[str, Any]:
    """Return fixture metadata only; phase 2 must not inspect task media."""
    if not step.startswith("fixture."):
        raise ValueError("fake provider only supports fixture steps")
    return {
        "provider": "fixture",
        "step": step,
    }


def _require_response(value: Any, operation: str) -> Any:
    if value is None or isinstance(value, Error):
        raise RuntimeError(f"worker API {operation} failed")
    return value


def _heartbeat_loop(
    client: AuthenticatedClient,
    job_id: int,
    lease_token: str,
    interval: float,
    stop: threading.Event,
    errors: list[Exception],
) -> None:
    while not stop.wait(interval):
        try:
            _require_response(
                heartbeat_job(
                    client=client,
                    id=job_id,
                    x_worker_lease_token=lease_token,
                ),
                "heartbeat",
            )
        except (httpx.HTTPError, RuntimeError) as exc:
            errors.append(exc)
            return


def run_once(
    client: AuthenticatedClient,
    capacity: int = 1,
    heartbeat_interval: float = 45.0,
    stop: threading.Event | None = None,
) -> bool:
    """Process at most one leased task. Return False when the queue is empty."""
    if heartbeat_interval <= 0:
        raise ValueError("heartbeat interval must be positive")
    _require_response(
        pulse(
            client=client,
            body=WorkerPulseInput(capacity=capacity, supported_steps=["fixture.*"]),
        ),
        "pulse",
    )
    if stop is not None and stop.is_set():
        return False
    claim = _require_response(claim_job(client=client), "claim")
    if claim is None or claim.job is None:
        return False
    job = claim.job
    lease_token = claim.lease_token
    heartbeat_stop = threading.Event()
    heartbeat_errors: list[Exception] = []
    heartbeater = threading.Thread(
        target=_heartbeat_loop,
        args=(
            client,
            job.id,
            lease_token,
            heartbeat_interval,
            heartbeat_stop,
            heartbeat_errors,
        ),
        daemon=True,
    )
    heartbeater.start()
    try:
        _require_response(
            get_job_input(
                client=client,
                id=job.id,
                x_worker_lease_token=lease_token,
            ),
            "task input",
        )
        result = fake_provider(job.step)
        heartbeat_stop.set()
        heartbeater.join()
        if heartbeat_errors:
            raise RuntimeError("worker lease lost")
        result_model = WorkerResultInputResult()
        result_model.additional_properties.update(result)
        _require_response(
            complete_job(
                client=client,
                id=job.id,
                body=WorkerResultInput(
                    input_version=job.input_version, result=result_model
                ),
                x_worker_lease_token=lease_token,
            ),
            "complete",
        )
    except (httpx.HTTPError, KeyError, RuntimeError, TypeError, ValueError):
        heartbeat_stop.set()
        heartbeater.join()
        try:
            _require_response(
                fail_job(
                    client=client,
                    id=job.id,
                    body=WorkerFailureInput(error="provider_unavailable"),
                    x_worker_lease_token=lease_token,
                ),
                "fail",
            )
        except (httpx.HTTPError, RuntimeError):
            # The lease may have expired or moved while the provider was busy.
            # In that case the old attempt must not mutate the recovered task.
            pass
    return True


def _worker_loop(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float,
    heartbeat_interval: float,
) -> None:
    while not stop.is_set():
        try:
            completed = run_once(client, capacity, heartbeat_interval, stop)
        except (httpx.HTTPError, RuntimeError):
            # Network outages are retried on the next poll. Do not log exception
            # details because HTTP client errors can include sensitive URLs.
            completed = False
        if not completed:
            stop.wait(poll_interval)


def run_until_stopped(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float = 5.0,
    heartbeat_interval: float = 45.0,
) -> None:
    """Run a bounded pool; stop claiming on shutdown and drain active tasks."""
    if capacity < 1 or capacity > 4:
        raise ValueError("worker capacity must be an integer from 1 to 4")
    if poll_interval <= 0 or heartbeat_interval <= 0:
        raise ValueError("worker intervals must be positive")

    with ThreadPoolExecutor(max_workers=capacity) as executor:
        workers = [
            executor.submit(
                _worker_loop,
                client,
                capacity,
                stop,
                poll_interval,
                heartbeat_interval,
            )
            for _ in range(capacity)
        ]
        while not stop.wait(0.1):
            pass
        for worker in workers:
            worker.result()


def main() -> None:
    api_url = os.environ.get("RECEIPTS_API_URL", "").strip()
    token = os.environ.get("RECEIPTS_WORKER_TOKEN", "").strip()
    capacity_text = os.environ.get("RECEIPTS_WORKER_CAPACITY", "1")
    if (
        not api_url
        or urlparse(api_url).scheme != "https"
        or not urlparse(api_url).netloc
    ):
        raise SystemExit("RECEIPTS_API_URL must be an HTTPS URL")
    if not token:
        raise SystemExit("RECEIPTS_WORKER_TOKEN is required")
    try:
        capacity = int(capacity_text)
    except ValueError as exc:
        raise SystemExit(
            "RECEIPTS_WORKER_CAPACITY must be an integer from 1 to 4"
        ) from exc
    if capacity < 1 or capacity > 4:
        raise SystemExit("RECEIPTS_WORKER_CAPACITY must be an integer from 1 to 4")
    ca_bundle = os.environ.get("RECEIPTS_WORKER_CA_BUNDLE")
    tls_context = ssl.create_default_context(cafile=ca_bundle) if ca_bundle else True
    with AuthenticatedClient(
        base_url=api_url.rstrip("/"),
        token=token,
        verify_ssl=tls_context,
        timeout=httpx.Timeout(30.0),
    ) as client:
        stop = threading.Event()

        def request_stop(_signum: int, _frame: object) -> None:
            stop.set()

        signal.signal(signal.SIGTERM, request_stop)
        signal.signal(signal.SIGINT, request_stop)
        run_until_stopped(client, capacity, stop)


if __name__ == "__main__":
    main()
