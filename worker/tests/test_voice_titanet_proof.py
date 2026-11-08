"""Operator CLI routing only; inference is covered by the adapter's real-model tests."""
import contextlib
import io
import json
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import patch

from snitch_worker import voice_titanet, voice_titanet_proof


class TitaNetProofTests(unittest.TestCase):
    def invoke(self, release, device="auto", cuda=False, refusal=None):
        with tempfile.TemporaryDirectory() as directory:
            manifest = Path(directory) / "manifest.json"
            manifest.write_text(json.dumps(release))
            output = io.StringIO()
            torch = types.SimpleNamespace(cuda=types.SimpleNamespace(is_available=lambda: cuda))
            with patch.dict("sys.modules", {"torch": torch}), patch.object(
                voice_titanet, "probe", return_value={"dimensions": 192}, side_effect=refusal
            ) as probe, contextlib.redirect_stdout(output):
                status = voice_titanet_proof.main([directory, "--device", device, "--release", str(manifest)])
            return status, json.loads(output.getvalue()), probe

    def test_cpu_release_uses_cpu_even_on_a_cuda_machine(self):
        status, result, probe = self.invoke(
            {"engine": "nemo-titanet", "version": "cpu-r1", "cuda": "cpu"}, cuda=True
        )
        self.assertEqual(status, 0)
        self.assertEqual(probe.call_args.args[1], "cpu")
        self.assertEqual(result["release_version"], "cpu-r1")
        self.assertTrue(result["ok"])

    def test_explicit_cuda_without_a_gpu_refuses_before_loading_model(self):
        status, result, probe = self.invoke(
            {"engine": "nemo-titanet", "version": "cuda-r1", "cuda": "12.8"}, device="cuda"
        )
        self.assertEqual((status, result), (1, {"ok": False, "refused": "cuda_unavailable"}))
        probe.assert_not_called()

    def test_cuda_release_cannot_fall_back_to_cpu(self):
        for device in ("auto", "cpu"):
            with self.subTest(device=device):
                status, result, probe = self.invoke(
                    {"engine": "nemo-titanet", "version": "cuda-r1", "cuda": "12.8"}, device=device
                )
                self.assertEqual((status, result["refused"]), (1, "cuda_unavailable"))
                probe.assert_not_called()

    def test_cuda_release_uses_gpu_and_reports_release_identity(self):
        status, result, probe = self.invoke(
            {"engine": "nemo-titanet", "version": "cuda-r1", "cuda": "12.8"}, cuda=True
        )
        self.assertEqual(status, 0)
        self.assertEqual(probe.call_args.args[1], "cuda")
        self.assertEqual(result["release_version"], "cuda-r1")

    def test_adapter_refusal_is_machine_readable(self):
        status, result, _ = self.invoke(
            {"engine": "nemo-titanet", "version": "cpu-r1", "cuda": "cpu"},
            refusal=voice_titanet.TitaNetRefused("unsupported_revision"),
        )
        self.assertEqual((status, result), (1, {"ok": False, "refused": "unsupported_revision"}))
