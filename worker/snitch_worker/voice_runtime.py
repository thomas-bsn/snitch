"""Select the locally validated runtime bound to a model, with legacy fallback."""
from __future__ import annotations

import json
import os
import re
from pathlib import Path
import subprocess
import tempfile
import threading
from typing import Any

from snitch_worker import cancellation, engine_runtime, voice_manifest
from snitch_worker.voice import VoiceProviderUnavailable

# The highest GPU memory peak an engine subprocess reported to the calling thread, the thread of a job; a benchmark
# task resets it and reports it (#403).
gpu_peak = threading.local()


def inference_device(manifest: dict[str, Any], environ: Any, gpu: Any = None) -> str | None:
    """The device an approved runtime may use, or a stable refusal before any inference starts.

    A CPU runtime is its own approved release and runs on the CPU. A CUDA release needs its GPU and the VRAM it
    declares: when the free VRAM is short the job fails with `gpu_out_of_memory` (it never degrades silently). CPU execution requires a distinct approved CPU release. Nothing is installed.
    When GPU telemetry is unavailable, force CUDA so the adapter refuses a missing GPU.
    """
    if manifest["cuda"] == "cpu":
        return "cpu"
    if manifest["cuda"] == "":
        # On-demand engine (ADR 0056): runs with or without a GPU. The adapter picks the device (auto); a GPU
        # that is present must still hold the VRAM the family declares.
        state = (gpu or engine_runtime.gpu_state)()
        if state is not None and state[0] and state[2] < manifest["vram_bytes"]:
            raise VoiceProviderUnavailable("gpu_out_of_memory")
        return None
    state = (gpu or engine_runtime.gpu_state)()
    if state is None:
        return "cuda"
    cuda, _total, free = state
    if not cuda:
        raise VoiceProviderUnavailable("cuda_unavailable")
    if cuda != manifest["cuda"]:
        raise VoiceProviderUnavailable("cuda_unavailable")
    if free < manifest["vram_bytes"]:
        raise VoiceProviderUnavailable("gpu_out_of_memory")
    return "cuda"


def call(manifest: dict[str, Any], path: str, environ: Any, **request: Any) -> dict[str, Any]:
    engine_runtime.validate_runtime(manifest)
    if not engine_runtime.installed(manifest, environ):
        raise VoiceProviderUnavailable("ml_libraries_missing")
    device = inference_device(manifest, environ)
    runtime_env = engine_runtime.environment(environ)
    if device is not None:
        runtime_env["SNITCH_VOICE_DEVICE"] = device
    python = str(engine_runtime.location(manifest, environ) / "bin/python")
    with tempfile.TemporaryDirectory(prefix="snitch-inference-") as temporary:
        incoming, outgoing = Path(temporary) / "in.json", Path(temporary) / "out.json"
        incoming.write_text(json.dumps({"manifest": manifest, "path": path, "manifest_path": voice_manifest.manifest_path(environ), **request}, allow_nan=False))
        # Only the Worker ships adapter code. The release cannot choose a module or search path.
        bootstrap = "import sys,runpy;sys.path.insert(0," + repr(str(Path(__file__).resolve().parent.parent)) + ");runpy.run_module('snitch_worker.engine_bridge',run_name='__main__')"
        try:
            cancellation.run_process([python, "-I", "-c", bootstrap, str(incoming), str(outgoing)],
                                     timeout=1800, env=runtime_env,
                                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if outgoing.stat().st_size > 32 << 20:
                raise ValueError()
            answer = json.loads(outgoing.read_text())
            if "error" in answer:
                safe = {"ml_libraries_missing", "cuda_unavailable", "gpu_out_of_memory", "model_load_failed", "inference_failed", "model_path_missing", "device_invalid"}
                refused = VoiceProviderUnavailable(answer["error"] if answer["error"] in safe else "inference_failed")
                detail = answer.get("detail")
                if isinstance(detail, str) and re.fullmatch(r"[a-z][a-z0-9_]{0,47}", detail):
                    refused.detail = detail  # stable adapter refusal code, kept for the Worker log
                raise refused
            if not isinstance(answer["result"], dict):
                raise ValueError()
            if isinstance(answer.get("gpu_peak_bytes"), int):
                gpu_peak.bytes = max(getattr(gpu_peak, "bytes", None) or 0, answer["gpu_peak_bytes"])
            return answer["result"]
        except (OSError, ValueError, KeyError, subprocess.SubprocessError):
            raise VoiceProviderUnavailable("inference_failed") from None


class RuntimeProvider:
    def __init__(self, legacy: Any) -> None:
        self.legacy = legacy

    def _binding(self, config: dict[str, Any], kind: str) -> tuple[str, dict[str, Any] | None]:
        path = (self.legacy._embedding_location(config) if kind == "fingerprint" else self.legacy._diarization_path(config))
        try:
            manifest = engine_runtime.model_binding(path)
            if manifest is not None:
                expected = (config.get("fingerprint_model") or {}).get("engine", "pyannote") if kind == "fingerprint" else config.get("provider")
                # A family engine is named after its package, not after the adapter of the catalogue: the binding of the
                # model and its kind decide (ADR 0056). A signed release is still matched by engine name.
                named = engine_runtime.is_on_demand(manifest) or manifest["engine"] == expected
                if not named or manifest["kind"] != kind or not engine_runtime.installed(manifest, self.legacy._environ):
                    raise VoiceProviderUnavailable("ml_libraries_missing")
            if manifest is None and getattr(self.legacy, "engine", "pyannote") in engine_runtime.SPEAKER_ENGINES:
                raise VoiceProviderUnavailable("ml_libraries_missing")
            return path, manifest
        except engine_runtime.RuntimeFailed:
            raise VoiceProviderUnavailable("model_load_failed") from None

    def ensure_diarization_ready(self, config: dict[str, Any]) -> None:
        _, manifest = self._binding(config, "diarization")
        if manifest is None:
            self.legacy.ensure_diarization_ready(config)

    def ensure_fingerprint_ready(self, config: dict[str, Any]) -> None:
        self.legacy._embedding_identity(config)
        _, manifest = self._binding(config, "fingerprint")
        if manifest is None:
            self.legacy.ensure_fingerprint_ready(config)

    def ensure_ready(self, config: dict[str, Any]) -> None:
        self.ensure_diarization_ready(config)
        self.ensure_fingerprint_ready(config)

    def diarize(self, config: dict[str, Any], audio: str, stream: int) -> dict[str, Any]:
        path, manifest = self._binding(config, "diarization")
        if manifest is None:
            return self.legacy.diarize(config, audio, stream)
        return call(manifest, path, self.legacy._environ, operation="infer", config=config, audio=audio, stream=stream)

    def fingerprint(self, config: dict[str, Any], diarization: Any, audio: str, stream: int) -> dict[str, Any]:
        path, manifest = self._binding(config, "fingerprint")
        if manifest is None:
            # New adapters have no bundled legacy runtime.
            if getattr(self.legacy, "engine", "pyannote") != "pyannote":
                raise VoiceProviderUnavailable("ml_libraries_missing")
            return self.legacy.fingerprint(config, diarization, audio, stream)
        return call(manifest, path, self.legacy._environ, operation="infer", config=config, diarization=diarization, audio=audio, stream=stream)
