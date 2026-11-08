"""Fixed Worker protocol executed with an approved runtime's Python, offline.

This module contains the adapter code; a registry release never supplies code or
an import path. Model metadata is data, not a HyperPyYAML/Hydra program.
"""
from __future__ import annotations

import io
import json
import os
from pathlib import Path, PurePosixPath
import sys
import tarfile
from typing import Any

from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_pyannote import DecodedAudio, PyannoteProvider


def _titanet(path: str, device: str) -> Any:
    import torch
    import yaml
    from omegaconf import OmegaConf
    from nemo.collections.asr.models import EncDecSpeakerLabelModel

    archives = list(Path(path).glob("*.nemo"))
    if len(archives) != 1:
        raise VoiceProviderUnavailable("model_load_failed")
    with tarfile.open(archives[0], "r:*") as archive:
        members = []
        for member in archive:
            members.append(member)
            if len(members) > 1024 or member.size > 3 << 30:
                raise VoiceProviderUnavailable("model_load_failed")
        for member in members:
            name = PurePosixPath(member.name)
            if name.is_absolute() or ".." in name.parts or not (member.isfile() or member.isdir()):
                raise VoiceProviderUnavailable("model_load_failed")
        def read(name: str, limit: int) -> bytes:
            entries = [m for m in members if PurePosixPath(m.name).name == name and m.isfile()]
            if len(entries) != 1 or entries[0].size > limit:
                raise VoiceProviderUnavailable("model_load_failed")
            stream = archive.extractfile(entries[0])
            if stream is None:
                raise VoiceProviderUnavailable("model_load_failed")
            return stream.read(limit + 1)
        config = yaml.safe_load(read("model_config.yaml", 1 << 20))
        weights = read("model_weights.ckpt", 3 << 30)
    approved = {"preprocessor": "AudioToMelSpectrogramPreprocessor", "encoder": "ConvASREncoder", "decoder": "SpeakerDecoder"}
    clean: dict[str, Any] = {}
    # No resolvers, nested targets, loss, augmentor, dataset or arbitrary model class.
    def plain(value: Any, depth: int = 0) -> None:
        if depth > 20:
            raise VoiceProviderUnavailable("model_load_failed")
        if isinstance(value, dict):
            if any(str(k).startswith("_") or k in {"cls", "target"} for k in value):
                raise VoiceProviderUnavailable("model_load_failed")
            for v in value.values():
                plain(v, depth + 1)
        elif isinstance(value, list):
            for v in value:
                plain(v, depth + 1)
        elif isinstance(value, str) and "${" in value:
            raise VoiceProviderUnavailable("model_load_failed")
        elif value is not None and not isinstance(value, (str, int, float, bool)):
            raise VoiceProviderUnavailable("model_load_failed")
    for name, cls in approved.items():
        part = dict(config[name])
        target = "nemo.collections.asr.modules." + cls
        if part.pop("_target_", None) != target:
            raise VoiceProviderUnavailable("model_load_failed")
        plain(part)
        clean[name] = {"_target_": target, **part}
    model = EncDecSpeakerLabelModel(cfg=OmegaConf.create(clean))
    model.load_state_dict(torch.load(io.BytesIO(weights), map_location="cpu", weights_only=True), strict=True)
    return model.eval().to(device)


def _speaker_loader(engine: str, path: str, device: str) -> Any:
    if engine == "speechbrain-ecapa":
        # Pinned files, data-only YAML scan, fixed architecture: see voice_ecapa (never from_hparams).
        from snitch_worker import voice_ecapa
        return voice_ecapa.speaker_loader(path, device)
    import torch
    import torchaudio

    if engine == "nemo-titanet":
        model = _titanet(path, device)
        def embed(waveform: Any) -> Any:
            _, vector = model(input_signal=waveform, input_signal_length=torch.tensor([waveform.shape[-1]], device=device))
            return vector
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
    Path(sys.argv[2]).write_text(json.dumps(result, allow_nan=False))


if __name__ == "__main__":
    main()
