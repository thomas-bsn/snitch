"""Fixed Worker protocol executed with an approved runtime's Python, offline.

This module contains the adapter code; a registry release never supplies code or
an import path. Model metadata is data, not a HyperPyYAML/Hydra program.
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import sys
from typing import Any

from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_pyannote import DecodedAudio, PyannoteProvider


def _speaker_loader(engine: str, path: str, device: str) -> Any:
    if engine == "speechbrain-ecapa":
        # Pinned files, data-only YAML scan, fixed architecture: see voice_ecapa (never from_hparams).
        from snitch_worker import voice_ecapa
        return voice_ecapa.speaker_loader(path, device)
    import torch
    import torchaudio

    if engine == "nemo-titanet":
        from snitch_worker import voice_titanet
        # Fixed loader: archive and config are data, weights are read with weights_only (see voice_titanet).
        embed = voice_titanet.load(path, device).embed
    else:
        raise VoiceProviderUnavailable("model_load_failed")
    def open_audio(audio_path: str) -> DecodedAudio:
        waveform, rate = torchaudio.load(audio_path)
        channels = waveform.shape[0]
        waveform = waveform.mean(dim=0, keepdim=True)
        if rate != 16000:
            waveform = torchaudio.functional.resample(waveform, rate, 16000)
        waveform = waveform.to(device)
        def window(start: float, end: float) -> list[float]:
            with torch.inference_mode():
                return embed(waveform[:, int(start * 16000):int(end * 16000)]).detach().cpu().reshape(-1).tolist()
        return DecodedAudio(window, 16000, channels, waveform.shape[-1])
    return open_audio


class SpeakerProvider(PyannoteProvider):
    def __init__(self, engine: str, **kwargs: Any) -> None:
        self.engine = engine
        super().__init__(load_embedder=lambda p, d: _speaker_loader(engine, p, d), **kwargs)

    def _embedding_identity(self, config: dict[str, Any] | None = None) -> tuple[str, str]:
        ref = (config or {}).get("fingerprint_model")
        if isinstance(ref, dict) and ref.get("engine") == self.engine:
            # Reuse the pinned identity and local-path validation, independently of diarization.
            return super()._embedding_identity({"fingerprint_model": {**ref, "engine": "pyannote"}})
        return super()._embedding_identity(config)


def execute(request: dict[str, Any]) -> dict[str, Any]:
    from snitch_worker import voice_manifest, voice_selftest
    from snitch_worker.voice_nemo import NemoProvider

    manifest, path = request["manifest"], request["path"]
    engine, kind = manifest["engine"], manifest["kind"]
    env = dict(os.environ)
    env["SNITCH_WORKER_MODELS_DIR"] = str(Path(request["manifest_path"]).parent)
    env[voice_manifest.EMBEDDING_PATH_ENV if kind == "fingerprint" else voice_manifest.DIARIZATION_PATH_ENV] = path
    env["SNITCH_VOICE_EMBEDDING_MODEL_VERSION"] = "runtime-selftest"
    provider = (SpeakerProvider(engine, environ=env) if engine in {"nemo-titanet", "speechbrain-ecapa"}
                else (NemoProvider if engine == "nemo" else PyannoteProvider)(environ=env))
    if request["operation"] == "verify":
        if kind == "fingerprint":
            voice_selftest.embed_synthetic(provider)
        else:
            voice_selftest.diarize_synthetic(provider, engine)
        return {}
    config = request["config"]
    if kind == "fingerprint":
        return provider.fingerprint(config, request["diarization"], request["audio"], request["stream"])
    return provider.diarize(config, request["audio"], request["stream"])


def main() -> None:
    try:
        request = json.loads(Path(sys.argv[1]).read_text())
        result = {"result": execute(request)}
    except Exception as exc:
        from snitch_worker.voice_pyannote import _classify
        result = {"error": exc.reason if isinstance(exc, VoiceProviderUnavailable) else _classify(exc, "model_load_failed")}
        if isinstance(getattr(exc, "detail", None), str):
            result["detail"] = exc.detail  # a stable refusal code of the adapter, never a library message
    Path(sys.argv[2]).write_text(json.dumps(result, allow_nan=False))


if __name__ == "__main__":
    main()
