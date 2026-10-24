"""Small HTTPS polling loop for the phase 2 fake provider."""

from __future__ import annotations

import hashlib
import os
import ssl
import time
from typing import Any
from urllib.parse import urljoin, urlparse

import httpx
from receipts_api.api.default.claim_worker_job import sync as claim_job
from receipts_api.api.default.complete_worker_job import sync as complete_job
from receipts_api.api.default.fail_worker_job import sync as fail_job
from receipts_api.api.default.get_worker_job_input import sync as get_job_input
from receipts_api.api.default.worker_pulse import sync as pulse
from receipts_api.client import AuthenticatedClient
from receipts_api.models.error import Error
from receipts_api.models.worker_failure_input import WorkerFailureInput
from receipts_api.models.worker_pulse_input import WorkerPulseInput
from receipts_api.models.worker_result_input import WorkerResultInput
from receipts_api.models.worker_result_input_result import WorkerResultInputResult


def fake_provider(media: bytes, step: str) -> dict[str, Any]:
    """Return deterministic synthetic output without interpreting media."""
    return {"provider": "fixture", "step": step, "media_sha256": hashlib.sha256(media).hexdigest()}


def _require_response(value: Any, operation: str) -> Any:
    if value is None or isinstance(value, Error):
        raise RuntimeError(f"worker API {operation} failed")
    return value


def run_once(client: AuthenticatedClient, capacity: int = 1) -> bool:
    """Process at most one leased task. Return False when the queue is empty."""
    _require_response(
        pulse(client=client, body=WorkerPulseInput(capacity=capacity)), "pulse"
    )
    claim = _require_response(claim_job(client=client), "claim")
    if claim is None or claim.job is None:
        return False
    job = claim.job
    try:
        task_input = _require_response(
            get_job_input(client=client, id=job.id), "task input"
        )
        parsed = task_input.to_dict()
        media_url = parsed["media_url"]
        url = urljoin(client._base_url.rstrip("/") + "/", media_url)
        if urlparse(url).scheme != "https" or urlparse(url).netloc != urlparse(client._base_url).netloc:
            raise ValueError("task media URL must use the configured HTTPS API origin")
        media_response = client.get_httpx_client().get(url)
        media_response.raise_for_status()
        result = fake_provider(media_response.content, job.step)
        result_model = WorkerResultInputResult()
        result_model.additional_properties.update(result)
        _require_response(
            complete_job(
                client=client,
                id=job.id,
                body=WorkerResultInput(input_version=job.input_version, result=result_model),
            ),
            "complete",
        )
    except (httpx.HTTPError, KeyError, RuntimeError, TypeError, ValueError):
        _require_response(
            fail_job(
                client=client,
                id=job.id,
                body=WorkerFailureInput(error="provider_unavailable"),
            ),
            "fail",
        )
    return True


def main() -> None:
    api_url = os.environ.get("RECEIPTS_API_URL", "").strip()
    token = os.environ.get("RECEIPTS_WORKER_TOKEN", "").strip()
    capacity_text = os.environ.get("RECEIPTS_WORKER_CAPACITY", "1")
    if not api_url or urlparse(api_url).scheme != "https" or not urlparse(api_url).netloc:
        raise SystemExit("RECEIPTS_API_URL must be an HTTPS URL")
    if not token:
        raise SystemExit("RECEIPTS_WORKER_TOKEN is required")
    try:
        capacity = int(capacity_text)
    except ValueError as exc:
        raise SystemExit("RECEIPTS_WORKER_CAPACITY must be an integer from 1 to 4") from exc
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
        while True:
            if not run_once(client, capacity):
                time.sleep(5)


if __name__ == "__main__":
    main()
