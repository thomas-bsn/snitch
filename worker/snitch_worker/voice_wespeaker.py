"""Generic fingerprint adapter of the WeSpeaker family (#375, ADR 0056): one adapter for every WeSpeaker
speaker-embedding model published as an ONNX graph (ResNet293-LM, ResNet34-LM, ...), run by the on-demand engine.

Fixed safeguards, outside any model's control:
  * no code of the model repository is imported or executed (Python files next to the weights are ignored; a
    configuration that asks for remote code is refused);
  * the weights are read only as an ONNX graph, a data format: a pickled checkpoint (.pt, .bin, .ckpt...) is refused;
  * ``config.yaml`` is read as data (``yaml.safe_load``, a closed set of numeric fields in bounded ranges);
  * nothing is downloaded: the engine never receives a Hugging Face token and the graph path is given by the Worker.

The libraries (torchaudio for audio and fbank, wespeakerruntime for inference) are reached through one small
backend, so they are imported only inside the engine's environment and are replaced by a fake in tests. Every
refusal is ``model_load_failed`` with a stable ``detail`` code; the text of a library error never travels.

Provenance: the distribution installed by the engine is ``wespeakerruntime`` (ONNX inference, depends on
torchaudio, onnxruntime, kaldiio); ``wespeaker`` itself is not published on PyPI. Features follow
wespeakerruntime's own pipeline: 16 kHz mono, samples scaled by 2**15, Kaldi fbank (hamming, no energy, no dither),
cepstral mean normalisation, no variance normalisation.
"""
from __future__ import annotations

import math
import re
from collections.abc import Callable
from pathlib import Path
from typing import Any

from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_pyannote import DecodedAudio

SAMPLE_RATE = 16000
MAX_WEIGHTS_BYTES = 2 << 30
MAX_CONFIG_BYTES = 1 << 20
MAX_DIMENSIONS = 4096
MAX_WINDOW_SECONDS = 120.0
DEFAULT_FBANK = {"num_mel_bins": 80, "frame_length": 25, "frame_shift": 10}
_RANGES = {"num_mel_bins": (20, 256), "frame_length": (10, 100), "frame_shift": (5, 50)}
# Checkpoints that load through pickle, and a safe format this adapter does not run.
_UNSAFE_SUFFIXES = (".pt", ".pth", ".bin", ".ckpt", ".pkl", ".pickle", ".npy", ".npz")
_UNSUPPORTED_SUFFIXES = (".safetensors",)
_REMOTE_CODE = re.compile(r"auto_map|trust_remote_code|custom_pipelines|!!?python", re.IGNORECASE)


class WeSpeakerRefused(VoiceProviderUnavailable):
    """The model cannot be used. ``reason`` stays the stable ``model_load_failed``; ``detail`` says why."""

    def __init__(self, detail: str) -> None:
        super().__init__("model_load_failed")
        self.detail = detail


def find_weights(directory: str) -> Path:
    """The single ONNX graph of the model directory, after refusing anything that is not a plain data file."""
    root = Path(directory)
    try:
        entries = sorted(p for p in root.iterdir())
    except OSError:
        raise WeSpeakerRefused("weights_missing") from None
    graphs = [p for p in entries if p.name.lower().endswith(".onnx")]
    if not graphs:
        names = [p.name.lower() for p in entries]
        if any(n.endswith(_UNSAFE_SUFFIXES) for n in names):
            raise WeSpeakerRefused("weights_format_unsafe")
        if any(n.endswith(_UNSUPPORTED_SUFFIXES) for n in names):
            raise WeSpeakerRefused("weights_format_unsupported")
        raise WeSpeakerRefused("weights_missing")
    if len(graphs) != 1:
        raise WeSpeakerRefused("weights_ambiguous")
    graph = graphs[0]
    try:
        if graph.is_symlink() or not graph.is_file() or graph.stat().st_size > MAX_WEIGHTS_BYTES:
            raise WeSpeakerRefused("weights_unsafe")
    except OSError:
        raise WeSpeakerRefused("weights_unsafe") from None
    return graph


def _number(value: Any) -> bool:
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value)


def read_options(directory: str) -> dict[str, Any]:
    """The fbank options of ``config.yaml`` (defaults when there is none), read as data and bounded.

    Dither is always 0: the embedding of a window must not depend on a random draw."""
    path = Path(directory) / "config.yaml"
    options: dict[str, Any] = dict(DEFAULT_FBANK)
    options["dither"] = 0.0
    try:
        if not path.exists():
            return options
        if path.is_symlink() or not path.is_file() or path.stat().st_size > MAX_CONFIG_BYTES:
            raise WeSpeakerRefused("config_invalid")
        text = path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        raise WeSpeakerRefused("config_invalid") from None
    if _REMOTE_CODE.search(text):
        raise WeSpeakerRefused("remote_code_refused")
    try:
        import yaml  # noqa: PLC0415 - installed with the engine's libraries when available
    except ImportError:
        return options  # the text scan above already ran; the defaults are those of the published WeSpeaker models
    try:
        config = yaml.safe_load(text)
    except Exception:  # noqa: BLE001 - parser messages quote the file and never travel
        raise WeSpeakerRefused("config_invalid") from None
    if not isinstance(config, dict):
        raise WeSpeakerRefused("config_invalid")
    dataset = config.get("dataset_args", {})
    if not isinstance(dataset, dict):
        raise WeSpeakerRefused("config_invalid")
    rate = dataset.get("sample_rate", SAMPLE_RATE)
    if not _number(rate):
        raise WeSpeakerRefused("config_invalid")
    if rate != SAMPLE_RATE:
        raise WeSpeakerRefused("variant_unsupported")
    fbank = dataset.get("fbank_args", {})
    if not isinstance(fbank, dict):
        raise WeSpeakerRefused("config_invalid")
    for key, (low, high) in _RANGES.items():
        if key in fbank:
            value = fbank[key]
            if not _number(value) or not low <= value <= high:
                raise WeSpeakerRefused("config_invalid")
            options[key] = int(value) if key == "num_mel_bins" else value
    return options


class Backend:
    """The libraries of the engine's environment. Imported lazily: the Worker itself runs without them."""

    def __init__(self) -> None:
        try:
            import torch  # noqa: PLC0415
            import torchaudio  # noqa: PLC0415
            import torchaudio.compliance.kaldi as kaldi  # noqa: PLC0415
        except ImportError:
            raise VoiceProviderUnavailable("ml_libraries_missing") from None
        self._torch, self._torchaudio, self._kaldi = torch, torchaudio, kaldi

    def decode(self, path: str) -> tuple[Any, int, int, int]:
        waveform, rate = self._torchaudio.load(path)
        channels = int(waveform.shape[0])
        if not bool(self._torch.isfinite(waveform).all()):
            raise ValueError("not finite")
        waveform = waveform.mean(dim=0, keepdim=True)
        if rate != SAMPLE_RATE:
            waveform = self._torchaudio.functional.resample(waveform, rate, SAMPLE_RATE)
        return waveform, SAMPLE_RATE, channels, int(waveform.shape[-1])

    def fbank(self, audio: Any, start: int, end: int, options: dict[str, Any]) -> Any:
        matrix = self._kaldi.fbank(audio[:, start:end] * (1 << 15), sample_frequency=SAMPLE_RATE, window_type="hamming", use_energy=False, **options)
        matrix = matrix.numpy()
        return (matrix - matrix.mean(axis=0))[None]  # CMN without CVN, batch of one: [1, frames, bins]

    def session(self, onnx_path: str) -> Callable[[Any], list[float]]:
        try:
            import wespeakerruntime  # noqa: PLC0415
        except ImportError:
            raise VoiceProviderUnavailable("ml_libraries_missing") from None
        speaker = wespeakerruntime.Speaker(onnx_path=onnx_path)  # a given path is never downloaded
        return lambda features: speaker.extract_embedding_feat(features, cmn=False)[0].tolist()


def _backend() -> Backend:
    return Backend()


def speaker_loader(directory: str, device: str, backend: Any = None) -> Callable[[str], DecodedAudio]:
    """Same shape as the other embedders: ``open_audio(path)`` returns a :class:`DecodedAudio`.

    ``device`` is accepted for the common interface; the ONNX graph runs on the CPU of the engine."""
    graph = find_weights(directory)
    options = read_options(directory)
    backend = backend or _backend()
    try:
        run = backend.session(str(graph))
    except VoiceProviderUnavailable:
        raise
    except Exception:  # noqa: BLE001 - a graph that does not load is a refusal; the library text never travels
        raise WeSpeakerRefused("weights_incompatible") from None

    def open_audio(audio_path: str) -> DecodedAudio:
        try:
            audio, rate, channels, samples = backend.decode(audio_path)
        except VoiceProviderUnavailable:
            raise
        except Exception:  # noqa: BLE001
            raise VoiceProviderUnavailable("inference_failed") from None

        def window(start: float, end: float) -> list[float]:
            end = min(end, start + MAX_WINDOW_SECONDS)
            try:
                vector = run(backend.fbank(audio, int(start * rate), int(end * rate), options))
                vector = [float(x) for x in vector]
            except VoiceProviderUnavailable:
                raise
            except Exception:  # noqa: BLE001 - library messages may carry paths
                raise VoiceProviderUnavailable("inference_failed") from None
            if not 1 <= len(vector) <= MAX_DIMENSIONS or not all(math.isfinite(x) for x in vector):
                raise VoiceProviderUnavailable("inference_failed")
            return vector

        return DecodedAudio(window, rate, channels, samples)

    return open_audio
