"""TitaNet speaker embeddings (NeMo), loaded from a ``.nemo`` archive without running any repository code.

The model directory holds exactly one ``.nemo`` file: a tar archive with ``model_config.yaml`` and
``model_weights.ckpt``. Only the architecture below is supported; the config is read as data (never as a
Hydra/OmegaConf program), the three modules are built directly from a fixed class list and the weights are
read with ``torch.load(weights_only=True)``, so that a tampered archive cannot execute anything.

Supported (``SUPPORTED_*``): 16 kHz, 80-band mel features, ``ConvASREncoder`` followed by a ``SpeakerDecoder``
(``attention`` or ``xvector`` pooling) with one integer embedding size. Anything else (QuartzNet/SpeakerNet,
ECAPA exported by NeMo, a different sample rate, ``safetensors`` or a bare ``.ckpt``...) is refused with
``model_load_failed`` and a stable ``detail`` code that the log carries.

The pure-Python part (archive and config validation) imports neither torch nor NeMo.
"""

from __future__ import annotations

import hashlib
import io
import json
import math
import os
import sys
import tarfile
import time
from pathlib import Path, PurePosixPath
from typing import Any

from snitch_worker.voice import VoiceProviderUnavailable

SAMPLE_RATE = 16000
CONFIG_NAME, WEIGHTS_NAME = "model_config.yaml", "model_weights.ckpt"
MAX_CONFIG_BYTES = 1 << 20
MAX_WEIGHTS_BYTES = 3 << 30
MAX_MEMBERS = 16

PREPROCESSOR = "nemo.collections.asr.modules.AudioToMelSpectrogramPreprocessor"
ENCODER = "nemo.collections.asr.modules.ConvASREncoder"
DECODER = "nemo.collections.asr.modules.SpeakerDecoder"
LOSSES = {"nemo.collections.asr.losses.angularloss.AngularSoftmaxLoss": True,
          "nemo.collections.common.losses.cross_entropy.CrossEntropyLoss": False}
SUPPORTED_POOL_MODES = ("attention", "xvector")
SUPPORTED_FEATURES = 80

_PREPROCESSOR_KEYS = {"sample_rate", "window_size", "window_stride", "window", "normalize", "n_fft", "preemph", "features",
                      "lowfreq", "highfreq", "log", "log_zero_guard_type", "log_zero_guard_value", "dither", "pad_to",
                      "frame_splicing", "exact_pad", "pad_value", "mag_power", "mel_norm"}
_ENCODER_KEYS = {"feat_in", "activation", "conv_mask", "jasper", "normalization_mode", "residual_mode", "norm_groups",
                 "frame_splicing", "init_mode"}
_BLOCK_KEYS = {"filters", "repeat", "kernel", "stride", "dilation", "dropout", "residual", "separable", "se",
               "se_context_size", "se_reduction_ratio", "kernel_size_factor", "stride_last", "residual_dense",
               "residual_panes", "groups", "heads"}
_DECODER_KEYS = {"feat_in", "num_classes", "emb_sizes", "pool_mode", "attention_channels", "init_mode"}


class TitaNetRefused(VoiceProviderUnavailable):
    """The model is not a supported TitaNet. ``reason`` stays the stable ``model_load_failed``; ``detail`` says why."""

    def __init__(self, detail: str) -> None:
        super().__init__("model_load_failed")
        self.detail = detail


def archive_of(path: str) -> str:
    """The single ``.nemo`` file of a model directory (or the file itself)."""
    if os.path.isfile(path):
        found = [path]
    else:
        try:
            found = sorted(str(p) for p in Path(path).iterdir() if p.is_file() and p.name.endswith(".nemo"))
        except OSError:
            raise TitaNetRefused("archive_missing") from None
    if not found:
        raise TitaNetRefused("archive_missing")
    if len(found) != 1 or not found[0].endswith(".nemo"):
        raise TitaNetRefused("archive_ambiguous")
    return found[0]


def read_archive(path: str) -> tuple[bytes, bytes]:
    """The config and the weights bytes of the model, after checking the archive is a plain, safe tar."""
    archive_path = archive_of(path)
    try:
        with tarfile.open(archive_path, "r:*") as archive:
            members = []
            for member in archive:
                members.append(member)
                if len(members) > MAX_MEMBERS:
                    raise TitaNetRefused("archive_unsafe")
            for member in members:
                name = PurePosixPath(member.name)
                if name.is_absolute() or ".." in name.parts or not (member.isfile() or member.isdir()):
                    raise TitaNetRefused("archive_unsafe")  # links, devices and escaping paths

            def read(name: str, limit: int, missing: str) -> bytes:
                entries = [m for m in members if PurePosixPath(m.name).name == name and m.isfile()]
                if not entries:
                    raise TitaNetRefused(missing)
                if len(entries) != 1 or entries[0].size > limit:
                    raise TitaNetRefused("archive_unsafe")
                stream = archive.extractfile(entries[0])
                if stream is None:
                    raise TitaNetRefused(missing)
                return stream.read(limit + 1)

            return read(CONFIG_NAME, MAX_CONFIG_BYTES, "config_missing"), read(WEIGHTS_NAME, MAX_WEIGHTS_BYTES, "weights_missing")
    except TitaNetRefused:
        raise
    except (tarfile.TarError, OSError, EOFError):
        raise TitaNetRefused("archive_not_nemo") from None


def parse_config(raw: bytes) -> dict[str, Any]:
    """YAML as data (``safe_load``): no tag can construct an object."""
    try:
        import yaml  # noqa: PLC0415 - installed with NeMo; absent from the minimal Worker

        value = yaml.safe_load(raw)
    except ImportError:
        raise
    except Exception:  # noqa: BLE001 - a YAML error message may quote the file
        raise TitaNetRefused("config_unreadable") from None
    if not isinstance(value, dict):
        raise TitaNetRefused("config_unreadable")
    return value


def _plain(value: Any, depth: int = 0) -> None:
    """Only data: no interpolation, no private key, no nested target, no object."""
    if depth > 20:
        raise TitaNetRefused("config_executable")
    if isinstance(value, dict):
        for key, item in value.items():
            if not isinstance(key, str) or key.startswith("_") or key in {"cls", "target"}:
                raise TitaNetRefused("config_executable")
            _plain(item, depth + 1)
    elif isinstance(value, list):
        for item in value:
            _plain(item, depth + 1)
    elif isinstance(value, str):
        if "${" in value:
            raise TitaNetRefused("config_executable")
    elif value is not None and not isinstance(value, (int, float, bool)):
        raise TitaNetRefused("config_executable")


def validate_config(config: Any) -> dict[str, Any]:
    """The approved constructor arguments of a supported TitaNet, or a refusal with its detail."""
    if not isinstance(config, dict):
        raise TitaNetRefused("config_unreadable")
    expected = {"preprocessor": (PREPROCESSOR, _PREPROCESSOR_KEYS), "encoder": (ENCODER, _ENCODER_KEYS), "decoder": (DECODER, _DECODER_KEYS)}
    parts: dict[str, dict[str, Any]] = {}
    for name, (target, allowed) in expected.items():
        part = config.get(name)
        if not isinstance(part, dict):
            raise TitaNetRefused("variant_unsupported")
        part = dict(part)
        found = part.pop("_target_", None)
        if found != target:
            # Another class is either another architecture or a class nobody approved to run.
            raise TitaNetRefused("variant_unsupported" if isinstance(found, str) and found.startswith("nemo.collections.") else "config_executable")
        _plain(part)
        if set(part) - allowed:
            raise TitaNetRefused("variant_unsupported")
        parts[name] = part
    loss = config.get("loss")
    angular = False
    if loss is not None:
        target = loss.get("_target_") if isinstance(loss, dict) else None
        if target not in LOSSES:
            raise TitaNetRefused("variant_unsupported")  # read as a name only, never instantiated
        angular = LOSSES[target]
    pre, enc, dec = parts["preprocessor"], parts["encoder"], parts["decoder"]
    if pre.get("sample_rate", 16000) != SAMPLE_RATE:
        raise TitaNetRefused("sample_rate_unsupported")
    if pre.get("features", 64) != SUPPORTED_FEATURES or enc.get("feat_in") != SUPPORTED_FEATURES:
        raise TitaNetRefused("variant_unsupported")
    blocks = enc.get("jasper")
    if not isinstance(blocks, list) or not 1 <= len(blocks) <= 32 or not all(isinstance(b, dict) and set(b) <= _BLOCK_KEYS for b in blocks):
        raise TitaNetRefused("variant_unsupported")
    size = dec.get("emb_sizes")
    classes = dec.get("num_classes")
    if type(size) is not int or not 16 <= size <= 4096 or type(classes) is not int or classes < 1 or dec.get("pool_mode") not in SUPPORTED_POOL_MODES:
        raise TitaNetRefused("variant_unsupported")
    if not isinstance(dec.get("feat_in"), int):
        raise TitaNetRefused("variant_unsupported")
    return {"preprocessor": pre, "encoder": enc, "decoder": {**dec, "angular": angular}, "dimensions": size}


class TitaNet:
    """A loaded model: ``embed`` maps a mono 16 kHz waveform ``[1, samples]`` to one vector."""

    def __init__(self, modules: Any, device: str, dimensions: int, archive_sha256: str) -> None:
        self.modules, self.device, self.dimensions, self.archive_sha256 = modules, device, dimensions, archive_sha256

    def embed(self, waveform: Any) -> Any:
        import torch  # noqa: PLC0415

        if waveform.ndim != 2 or waveform.shape[0] != 1 or waveform.shape[-1] < 1 or not bool(torch.isfinite(waveform).all()):
            raise VoiceProviderUnavailable("inference_failed")
        length = torch.tensor([waveform.shape[-1]], device=self.device)
        with torch.inference_mode():
            features, feature_length = self.modules.preprocessor(input_signal=waveform.to(self.device), length=length)
            encoded = self.modules.encoder(audio_signal=features, length=feature_length)
            encoded, encoded_length = encoded if isinstance(encoded, tuple) else (encoded, None)
            _, vector = self.modules.decoder(encoder_output=encoded, length=encoded_length)
        if vector.shape[-1] != self.dimensions:
            raise VoiceProviderUnavailable("inference_failed")
        return vector


def load(path: str, device: str) -> TitaNet:  # needs NeMo and the model: exercised by the real-inference tests
    import torch
    from omegaconf import OmegaConf
    from nemo.collections.asr.modules import AudioToMelSpectrogramPreprocessor, ConvASREncoder, SpeakerDecoder

    raw_config, weights = read_archive(path)
    approved = validate_config(parse_config(raw_config))
    digest = hashlib.sha256(Path(archive_of(path)).read_bytes()).hexdigest()
    try:
        container = torch.nn.Module()
        container.preprocessor = AudioToMelSpectrogramPreprocessor(**approved["preprocessor"])
        container.encoder = ConvASREncoder(**{**approved["encoder"], "jasper": OmegaConf.create(approved["encoder"]["jasper"])})
        container.decoder = SpeakerDecoder(**approved["decoder"])
    except (TypeError, ValueError, KeyError):
        raise TitaNetRefused("variant_unsupported") from None
    try:
        state = torch.load(io.BytesIO(weights), map_location="cpu", weights_only=True)
        container.load_state_dict(state, strict=True)
    except Exception:  # noqa: BLE001 - unpickling errors and key mismatches both mean "not this model"
        raise TitaNetRefused("weights_incompatible") from None
    return TitaNet(container.eval().to(device), device, approved["dimensions"], digest)


def synthetic_waveform(seconds: float, seed: int = 1, rate: int = SAMPLE_RATE) -> list[float]:
    """Deterministic voiced-like signal (harmonics of a drifting pitch plus fixed-seed noise): no recorded audio."""
    import random

    rng = random.Random(seed)
    pitch, phase, samples = 110.0 + 20.0 * (seed % 5), 0.0, []
    for index in range(int(seconds * rate)):
        pitch_now = pitch * (1.0 + 0.05 * math.sin(2 * math.pi * 3.0 * index / rate))
        phase += 2 * math.pi * pitch_now / rate
        value = sum(math.sin(k * phase) / k for k in range(1, 12))
        envelope = 0.55 + 0.45 * math.sin(2 * math.pi * 2.0 * index / rate)
        samples.append(0.12 * envelope * value + 0.005 * rng.uniform(-1.0, 1.0))
    return samples


def probe(path: str, device: str, seconds: float = 6.0) -> dict[str, Any]:  # pragma: no cover - needs NeMo and the model
    """What the operator pastes back after a run on the target machine: identity, shape, norm, timings, memory."""
    import resource
    import torch

    started = time.monotonic()
    model = load(path, device)
    loaded = time.monotonic()
    wave = torch.tensor([synthetic_waveform(seconds)], dtype=torch.float32)
    vector = model.embed(wave).reshape(-1)
    done = time.monotonic()
    again = model.embed(wave).reshape(-1)
    report: dict[str, Any] = {
        "engine": "nemo-titanet", "archive_sha256": model.archive_sha256, "dimensions": int(vector.shape[0]),
        "norm_raw": float(vector.norm()), "norm_unit": float((vector / vector.norm()).norm()),
        "repeat_max_abs_diff": float((vector - again).abs().max()),
        "audio_seconds": seconds, "device": device, "load_ms": int((loaded - started) * 1000), "inference_ms": int((done - loaded) * 1000),
        "torch": torch.__version__, "torch_cuda": torch.version.cuda or "", "python": sys.version.split()[0],
        "max_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * (1 if sys.platform == "darwin" else 1024),
    }
    try:
        import nemo

        report["nemo"] = nemo.__version__
    except Exception:  # noqa: BLE001
        report["nemo"] = ""
    if device.startswith("cuda"):
        report["cuda_device"] = torch.cuda.get_device_name(0)
        report["vram_peak_bytes"] = int(torch.cuda.max_memory_allocated())
        report["vram_total_bytes"] = int(torch.cuda.get_device_properties(0).total_memory)
    return report


def main(argv: list[str] | None = None) -> int:  # pragma: no cover - needs NeMo and the model
    args = list(sys.argv[1:] if argv is None else argv)
    if not args or args[0] in {"-h", "--help"}:
        print("usage: voice_titanet <model directory or .nemo file> [cpu|cuda]", file=sys.stderr)
        return 2
    try:
        print(json.dumps(probe(args[0], args[1] if len(args) > 1 else "cpu"), indent=2, sort_keys=True))
    except TitaNetRefused as refused:
        print(json.dumps({"refused": refused.detail}), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
