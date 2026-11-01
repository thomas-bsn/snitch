"""A Worker deleted in Snitch cleans its own models volume and goes quiet (ADR 0021)."""

import os
import tempfile
import threading
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_client.models.error import Error
from snitch_client.models.get_worker_revocation_response_200 import GetWorkerRevocationResponse200

from snitch_worker import revocation, runner

TOKEN_INVALID = Error(error="worker_token_invalid")


def populated_models_dir(root: str) -> Path:
    models = Path(root)
    (models / "diarization").mkdir()
    (models / "diarization" / "config.yaml").write_text("synthetic")
    (models / "embedding").mkdir()
    (models / "embedding" / "pytorch_model.bin").write_bytes(b"synthetic")
    (models / ".snitch-voice-models.json").write_text("{}")
    (models / ".cache" / "huggingface" / "xet").mkdir(parents=True)
    (models / ".cache" / "huggingface" / "xet" / "chunk").write_bytes(b"synthetic")
    return models


class FakeClient:
    """Stands in for AuthenticatedClient: records whether the worker closed its HTTP client."""

    def __init__(self):
        self.closed = False

    def get_httpx_client(self):
        return self

    def close(self):
        self.closed = True


def run_pool(client, models_dir, stop):
    thread = threading.Thread(
        target=runner.run_until_stopped,
        args=(client, 1, stop),
        kwargs={"poll_interval": 0.01, "models_dir": models_dir},
    )
    thread.start()
    thread.join(3)
    return thread


class ConfirmedRevocationTests(unittest.TestCase):
    def test_confirmed_revocation_erases_the_models_volume_and_stops_calling_snitch(self):
        with tempfile.TemporaryDirectory() as root:
            models = populated_models_dir(root)
            client = FakeClient()
            stop = threading.Event()
            with (
                patch.object(runner, "pulse", return_value=TOKEN_INVALID) as pulse,
                patch.object(runner, "claim_job") as claim,
                patch.object(
                    revocation,
                    "fetch_revocation",
                    return_value=GetWorkerRevocationResponse200(revoked=True),
                ) as confirm,
            ):
                thread = run_pool(client, root, stop)

            self.assertFalse(thread.is_alive(), "the pool kept running after a confirmed revocation")
            self.assertTrue(stop.is_set())
            self.assertEqual(pulse.call_count, 1, "the worker called Snitch again after the confirmation")
            confirm.assert_called_once()
            claim.assert_not_called()
            self.assertTrue(client.closed, "the HTTP client must be closed so no later call can leave")
            self.assertEqual(sorted(p.name for p in models.iterdir()), [revocation.MARKER_NAME])
            self.assertTrue(revocation.is_revoked(root))


class UnconfirmedUnauthorizedTests(unittest.TestCase):
    def assert_nothing_erased(self, confirmation):
        with tempfile.TemporaryDirectory() as root:
            models = populated_models_dir(root)
            before = sorted(str(p.relative_to(models)) for p in models.rglob("*"))
            client = FakeClient()
            stop = threading.Event()
            calls = []

            def pulse(**_kwargs):
                calls.append(1)
                if len(calls) >= 3:
                    stop.set()
                return TOKEN_INVALID

            fetch = {"side_effect": confirmation} if isinstance(confirmation, Exception) else {"return_value": confirmation}
            with (
                patch.object(runner, "pulse", side_effect=pulse),
                patch.object(revocation, "fetch_revocation", **fetch),
            ):
                thread = run_pool(client, root, stop)

            self.assertFalse(thread.is_alive())
            self.assertGreaterEqual(len(calls), 3, "an unconfirmed 401 must keep the worker retrying")
            self.assertEqual(sorted(str(p.relative_to(models)) for p in models.rglob("*")), before)
            self.assertFalse(revocation.is_revoked(root))
            self.assertFalse(client.closed)

    def test_unknown_token_is_not_a_revocation(self):
        self.assert_nothing_erased(TOKEN_INVALID)

    def test_not_revoked_answer_erases_nothing(self):
        self.assert_nothing_erased(GetWorkerRevocationResponse200(revoked=False))

    def test_unreachable_confirmation_erases_nothing(self):
        import httpx

        self.assert_nothing_erased(httpx.ConnectError("synthetic outage"))


class RevokedMarkerAtStartupTests(unittest.TestCase):
    def test_marker_makes_the_worker_inert_without_any_network_call(self):
        with tempfile.TemporaryDirectory() as root:
            Path(root, revocation.MARKER_NAME).write_text("revoked\n")
            started = threading.Event()

            def inert(event):
                self.assertIsInstance(event, threading.Event)
                started.set()

            env = {
                "SNITCH_API_URL": "https://snitch.example",
                "SNITCH_WORKER_TOKEN": "rw_synthetic-token",
                "SNITCH_WORKER_MODELS_DIR": root,
            }
            with (
                patch.dict(os.environ, env),
                patch.object(runner, "AuthenticatedClient", side_effect=AssertionError("network client created")),
                patch.object(runner, "pulse", side_effect=AssertionError("Snitch called")),
                patch.object(revocation, "fetch_revocation", side_effect=AssertionError("Snitch called")),
                patch.object(runner.signal, "signal"),
                patch.object(revocation, "wait_inert", side_effect=inert),
            ):
                runner.main()
            self.assertTrue(started.is_set(), "a revoked worker must stay inert, not exit into a restart loop")

    def test_bootstrap_downloads_nothing_once_revoked(self):
        with tempfile.TemporaryDirectory() as root:
            Path(root, revocation.MARKER_NAME).write_text("revoked\n")
            launched = []
            from snitch_worker import bootstrap

            bootstrap.main(
                environ={
                    "SNITCH_WORKER_MODELS_DIR": root,
                    "SNITCH_VOICE_DOWNLOAD_ON_START": "1",
                    "SNITCH_VOICE_CONFIRM_TERMS": "1",
                    "HF_TOKEN": "hf_synthetic",
                },
                downloader=lambda **_kwargs: self.fail("a revoked worker downloaded models"),
                resolve_revision=lambda *_args: self.fail("a revoked worker contacted Hugging Face"),
                exec_worker=launched.append,
                log=lambda _line: None,
            )
            self.assertEqual(len(launched), 1, "the runner still starts, to stay inert under the restart policy")
            self.assertEqual(os.listdir(root), [revocation.MARKER_NAME])

    def test_valid_worker_without_marker_is_not_revoked(self):
        with tempfile.TemporaryDirectory() as root:
            populated_models_dir(root)
            self.assertFalse(revocation.is_revoked(root))


if __name__ == "__main__":
    unittest.main()
