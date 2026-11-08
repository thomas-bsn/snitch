"""Select the locally validated runtime bound to a model, with legacy fallback."""
from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import tempfile
from typing import Any

from snitch_worker import cancellation, engine_runtime, voice_manifest
from snitch_worker.voice import VoiceProviderUnavailable


def call(manifest: dict[str, Any], path: str, environ: Any, **request: Any) -> dict[str, Any]:
    engine_runtime.validate(manifest, manifest.get("engine", ""), manifest.get("kind", ""))
    if not engine_runtime.installed(manifest, environ):
        raise VoiceProviderUnavailable("ml_libraries_missing")
    python = str(engine_runtime.location(manifest, environ) / "bin/python")
    with tempfile.TemporaryDirectory(prefix="snitch-inference-") as temporary:
        incoming, outgoing = Path(temporary) / "in.json", Path(temporary) / "out.json"
        incoming.write_text(json.dumps({"manifest": manifest, "path": path, "manifest_path": voice_manifest.manifest_path(environ), **request}, allow_nan=False))
        # Only the Worker ships adapter code. The release cannot choose a module or search path.
        bootstrap = "import sys,runpy;sys.path.insert(0," + repr(str(Path(__file__).resolve().parent.parent)) + ");runpy.run_module('snitch_worker.engine_bridge',run_name='__main__')"
        try:
            cancellation.run_process([python, "-I", "-c", bootstrap, str(incoming), str(outgoing)],
                                     timeout=1800, env=engine_runtime.environment(environ),
                                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if outgoing.stat().st_size > 32 << 20:
                raise ValueError()
            answer = json.loads(outgoing.read_text())
            if "error" in answer:
                safe = {"ml_libraries_missing", "cuda_unavailable", "gpu_out_of_memory", "model_load_failed", "inference_failed", "model_path_missing", "device_invalid"}
                raise VoiceProviderUnavailable(answer["error"] if answer["error"] in safe else "inference_failed")
            if not isinstance(answer["result"], dict):
                raise ValueError()
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
                if manifest["engine"] != expected or manifest["kind"] != kind or not engine_runtime.installed(manifest, self.legacy._environ):
                    raise VoiceProviderUnavailable("ml_libraries_missing")
            if manifest is None and getattr(self.legacy, "engine", "pyannote") in {"nemo-titanet", "speechbrain-ecapa"}:
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
