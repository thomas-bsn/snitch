"""SepFormer voice separation of the on-demand SpeechBrain engine (#401, ADR 0065), loaded without executing model code.

A separation model isolates the voices of a mixture: one audio in, N sources out, N being what the model was
trained for. Nothing here belongs to the analysis of a clip: the adapter is called by the test inference of an
install order and by an explicit caller (``voice_separation.separate``), never by a production step.

SpeechBrain normally builds the model from ``hyperparams.yaml`` with HyperPyYAML, whose tags instantiate or call
arbitrary Python objects (``SepformerSeparation.from_hparams``). This adapter never runs that program. It:

1. checks the shape of the model directory (no code, no link, no directory, only the files it reads or knowingly
   ignores: the repositories also ship training leftovers such as ``optimizer.ckpt``, which are never opened);
2. parses ``hyperparams.yaml`` as data only and refuses every tag outside the allowlist of ``voice_ecapa``;
3. reads the limits of the model (sample rate, number of separated voices) and the SepFormer architecture from
   that file, against a closed list of classes and parameters, and refuses anything else;
4. scans the pickle opcodes of each checkpoint and only then calls ``torch.load(weights_only=True)``.

The network is built here from ``speechbrain.lobes.models.dual_path`` and those values. Every refusal raises
:class:`UnsupportedVariant` (stable reason ``model_load_failed``, a ``detail`` that is safe to log).
"""
from __future__ import annotations

import os
import wave
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Callable, Mapping

from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_ecapa import (_STANDARD_TAGS, IGNORED_FILES, MAX_CHANNELS, UnsupportedAudio, UnsupportedVariant, _plain,
                                       _yaml, check_audio_file, scan_checkpoint, scan_hyperparams)

__all__ = ["Separator", "load", "read_architecture", "resolve_device", "scan_hyperparams", "separate_file"]

ENGINE = "speechbrain-sepformer"
# Limits this adapter accepts; the limits of a given model are read from its own file.
SUPPORTED_SAMPLE_RATES = (8000, 16000)
MIN_SOURCES, MAX_SOURCES = 2, 3
# The longest window separated in one call: the dual-path network holds the whole window in memory. A longer one is
# refused, never cut silently.
MAX_WINDOW_SECONDS = 60.0
# The encoder needs a few frames; a shorter window has nothing to separate.
MIN_WINDOW_SECONDS = 0.1

CHECKPOINTS = {"encoder": "encoder.ckpt", "masknet": "masknet.ckpt", "decoder": "decoder.ckpt"}
REQUIRED_FILES = ("hyperparams.yaml", *CHECKPOINTS.values())
# Files the SpeechBrain repositories ship beside the inference files. They are never opened.
NEVER_READ = {"optimizer.ckpt", "brain.ckpt", "counter.ckpt", "lr_scheduler.ckpt", "dataloader-TRAIN.ckpt", "CKPT.yaml",
              "hyperparams_train.yaml", "config.son"}

_DUAL_PATH = "speechbrain.lobes.models.dual_path."
_BLOCK = {"num_layers": int, "d_model": int, "nhead": int, "d_ffn": int, "dropout": (int, float),
          "use_positional_encoding": bool, "norm_before": bool}
# YAML key -> (class, name in the result, parameters and their types). Nothing else is ever built.
_NODES: dict[str, tuple[str, str, dict[str, Any]]] = {
    "Encoder": ("Encoder", "encoder", {"kernel_size": int, "out_channels": int}),
    "SBtfintra": ("SBTransformerBlock", "intra", _BLOCK),
    "SBtfinter": ("SBTransformerBlock", "inter", _BLOCK),
    "MaskNet": ("Dual_Path_Model", "masknet", {"num_spks": int, "in_channels": int, "out_channels": int, "num_layers": int, "K": int,
                                                "norm": str, "linear_layer_after_inter_intra": bool, "skip_around_intra": bool}),
    "Decoder": ("Decoder", "decoder", {"in_channels": int, "out_channels": int, "kernel_size": int, "stride": int, "bias": bool}),
}
_MASK_REFERENCES = {"intra_model": "<SBtfintra>", "inter_model": "<SBtfinter>"}
_NORMS = {"ln", "gln", "cln", "bn"}
_MAX_SIZE = 8192  # no parameter of a SepFormer comes near; a larger one is not a model, it is an allocation


@dataclass(frozen=True)
class Separator:
    """A loaded separation model. ``separate`` takes a ``[1, samples]`` float tensor at ``sample_rate`` and returns
    a ``[num_sources, samples]`` tensor: always ``num_sources`` voices, whatever the mixture holds."""

    sample_rate: int
    num_sources: int
    separate: Callable[[Any], Any]


def resolve_device(environ: Mapping[str, str], probe: Callable[[], str] | None = None) -> str:
    """The device of the Worker setting (``SNITCH_VOICE_DEVICE``: auto, cuda or cpu). A GPU that was asked for and
    is absent is a refusal, never a silent fallback to the CPU."""
    from snitch_worker import voice_pyannote  # noqa: PLC0415

    wanted = environ.get(voice_pyannote.DEVICE_ENV, "auto").strip().lower() or "auto"
    if wanted not in {"auto", "cuda", "cpu"}:
        raise VoiceProviderUnavailable("device_invalid")
    if wanted == "cpu":
        return "cpu"
    available = (probe or voice_pyannote._probe_device)()
    if wanted == "cuda" and available != "cuda":
        raise VoiceProviderUnavailable("cuda_unavailable")
    return available


# -- files ---------------------------------------------------------------------------------------------------------

def verify_directory(directory: str | os.PathLike[str]) -> None:
    try:
        entries = {entry.name: entry for entry in Path(directory).iterdir()}
    except OSError:
        raise UnsupportedVariant("model_directory_unreadable") from None
    for name, entry in entries.items():
        if name == ".cache":
            continue  # huggingface_hub's download bookkeeping: never read
        if entry.is_symlink() or not entry.is_file():
            raise UnsupportedVariant("unexpected_directory_or_link")
        if name.endswith((".py", ".pyc", ".so", ".dll", ".sh")):
            raise UnsupportedVariant("code_file_in_model")  # no Hugging Face script, ever
    if not set(REQUIRED_FILES) <= set(entries):
        raise UnsupportedVariant("model_file_missing")
    extra = set(entries) - set(REQUIRED_FILES) - NEVER_READ - IGNORED_FILES - {".cache"}
    if any(not name.endswith(".wav") for name in extra):  # the example mixtures of the model cards are tolerated
        raise UnsupportedVariant("unexpected_file")


# -- hyperparams.yaml, as data ---------------------------------------------------------------------------------------

def _typed(value: Any, expected: Any) -> bool:
    kinds = expected if isinstance(expected, tuple) else (expected,)
    if type(value) not in kinds:  # a bool is not an int here
        return False
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return 0 <= value <= _MAX_SIZE
    return True


def read_architecture(root: Any) -> dict[str, Any]:
    """The limits and the SepFormer architecture a scanned ``hyperparams.yaml`` describes, as plain values.

    Each network node must name its dual-path class and carry exactly the known parameters with plain values of
    the expected type: a different class, a missing or an unknown parameter is another architecture."""
    yaml = _yaml()
    top = {key.value: value for key, value in root.value if isinstance(key, yaml.ScalarNode)}
    scalars = {name: _plain(node, {}) for name, node in top.items()
               if isinstance(node, yaml.ScalarNode) and node.tag in _STANDARD_TAGS}
    out: dict[str, Any] = {}
    for key, (cls, name, schema) in _NODES.items():
        node = top.get(key)
        if node is None or node.tag != "!new:" + _DUAL_PATH + cls or not isinstance(node, yaml.MappingNode):
            raise UnsupportedVariant("unsupported_architecture")
        kwargs: dict[str, Any] = {}
        for item, value in node.value:
            if not isinstance(item, yaml.ScalarNode):
                raise UnsupportedVariant("unsupported_architecture")
            if key == "MaskNet" and item.value in _MASK_REFERENCES:
                # The two transformer blocks are wired by this code; the file may only name them.
                if value.tag != "!ref" or not isinstance(value, yaml.ScalarNode) or value.value != _MASK_REFERENCES[item.value]:
                    raise UnsupportedVariant("unsupported_architecture")
                continue
            if item.value not in schema or item.value in kwargs:
                raise UnsupportedVariant("unsupported_architecture")
            kwargs[item.value] = _plain(value, scalars)
        if set(kwargs) != set(schema) or not all(_typed(kwargs[k], expected) for k, expected in schema.items()):
            raise UnsupportedVariant("unsupported_architecture")
        if key == "MaskNet" and {item.value for item, _ in node.value} & set(_MASK_REFERENCES) != set(_MASK_REFERENCES):
            raise UnsupportedVariant("unsupported_architecture")
        out[name] = kwargs
    rate, sources = scalars.get("sample_rate"), scalars.get("num_spks")
    if type(rate) is not int or rate not in SUPPORTED_SAMPLE_RATES:
        raise UnsupportedVariant("sample_rate_unsupported")
    if type(sources) is not int or not MIN_SOURCES <= sources <= MAX_SOURCES:
        raise UnsupportedVariant("source_count_unsupported")
    if out["masknet"]["num_spks"] != sources or out["decoder"]["out_channels"] != 1 or out["masknet"]["norm"] not in _NORMS:
        raise UnsupportedVariant("unsupported_architecture")
    out["sample_rate"], out["num_spks"] = rate, sources
    return out


# -- the model -------------------------------------------------------------------------------------------------------

def load(directory: str, device: str) -> Separator:
    """Verify, scan, then build the SepFormer the file describes and load its weights. Offline: nothing is fetched,
    no hub, no ``from_hparams``."""
    verify_directory(directory)
    root = Path(directory)
    try:
        text = (root / "hyperparams.yaml").read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        raise UnsupportedVariant("model_directory_unreadable") from None
    architecture = read_architecture(scan_hyperparams(text))
    for name in CHECKPOINTS.values():
        scan_checkpoint(root / name)
    try:
        import torch  # noqa: PLC0415 - lazy: the Worker runs without the ML stack
        from speechbrain.lobes.models import dual_path  # noqa: PLC0415
    except ImportError:
        raise VoiceProviderUnavailable("ml_libraries_missing") from None
    modules = {
        "encoder": dual_path.Encoder(**architecture["encoder"]),
        "masknet": dual_path.Dual_Path_Model(intra_model=dual_path.SBTransformerBlock(**architecture["intra"]),
                                             inter_model=dual_path.SBTransformerBlock(**architecture["inter"]), **architecture["masknet"]),
        "decoder": dual_path.Decoder(**architecture["decoder"]),
    }
    for name, module in modules.items():
        state = torch.load(root / CHECKPOINTS[name], map_location=device, weights_only=True)
        module.load_state_dict(state, strict=True)
        modules[name] = module.to(device).eval()
    encoder, masknet, decoder = modules["encoder"], modules["masknet"], modules["decoder"]
    sources = architecture["num_spks"]

    def separate(mixture: Any) -> Any:
        # The forward pass of SpeechBrain's SepformerSeparation.separate_batch: mask the encoded mixture once per
        # voice, decode each, and bring the result back to the length of the input.
        with torch.inference_mode():
            mixture = mixture.to(device)
            encoded = encoder(mixture)
            masked = torch.stack([encoded] * sources) * masknet(encoded)
            estimate = torch.cat([decoder(masked[index]).unsqueeze(-1) for index in range(sources)], dim=-1)
            length, got = mixture.shape[1], estimate.shape[1]
            if got < length:
                estimate = torch.nn.functional.pad(estimate, (0, 0, 0, length - got))
            return estimate[0, :length, :].transpose(0, 1)

    return Separator(architecture["sample_rate"], sources, separate)


def separate_file(separator: Separator, audio_path: str, output_dir: str,
                  windows: list[tuple[float | None, float | None]] | None = None) -> list[dict[str, Any]]:
    """Separate windows of an audio file and write each source as a mono 16-bit WAV in ``output_dir``.

    The file is decoded, mixed down to mono and resampled to the rate of the model once, then every window is cut
    from it. A window is ``(start, end)`` in seconds of the file, ``None`` for its edge (the whole file by default);
    it is cut to the file and refused when empty, too short or longer than MAX_WINDOW_SECONDS. Every window is
    checked before any is separated. The sources keep their relative levels: one common gain per window, applied
    only when a sample would clip. Returns, in the order asked, the window really separated and one entry per source
    (``window-<w>-source-<i>.wav``)."""
    import torch  # noqa: PLC0415
    import torchaudio  # noqa: PLC0415

    check_audio_file(audio_path)
    try:
        waveform, rate = torchaudio.load(audio_path)
    except Exception:  # noqa: BLE001 - library text never travels
        raise UnsupportedAudio("audio_undecodable") from None
    if not 1 <= int(waveform.shape[0]) <= MAX_CHANNELS:
        raise UnsupportedAudio("unsupported_channels")
    if not bool(torch.isfinite(waveform).all()):
        raise UnsupportedAudio("audio_not_finite")
    waveform = waveform.mean(dim=0, keepdim=True)
    if rate != separator.sample_rate:
        waveform = torchaudio.functional.resample(waveform, rate, separator.sample_rate)
    rate = separator.sample_rate
    total = int(waveform.shape[-1])
    spans = []
    for start, end in windows or [(None, None)]:
        first = 0 if start is None else max(0, int(round(float(start) * rate)))
        last = total if end is None else min(total, int(round(float(end) * rate)))
        if last <= first:
            raise UnsupportedAudio("window_empty")
        if last - first < MIN_WINDOW_SECONDS * rate:
            raise UnsupportedAudio("window_too_short")
        if last - first > MAX_WINDOW_SECONDS * rate:
            raise UnsupportedAudio("window_too_long")
        spans.append((first, last))
    os.makedirs(output_dir, exist_ok=True)
    results = []
    for number, (first, last) in enumerate(spans):
        sources = separator.separate(waveform[:, first:last]).detach().cpu()
        if tuple(sources.shape) != (separator.num_sources, last - first) or not bool(torch.isfinite(sources).all()):
            raise VoiceProviderUnavailable("inference_failed")
        peak = float(sources.abs().max())
        gain = 1.0 if peak <= 1.0 else 1.0 / peak
        pcm = (sources * gain * 32767.0).round().clamp(-32768, 32767).to(torch.int16)
        written = []
        for index in range(separator.num_sources):
            path = os.path.join(output_dir, f"window-{number}-source-{index}.wav")
            with wave.open(path, "wb") as out:
                out.setnchannels(1)
                out.setsampwidth(2)
                out.setframerate(rate)
                out.writeframes(pcm[index].numpy().tobytes())
            written.append({"index": index, "path": path, "samples": last - first})
        results.append({"sample_rate": rate, "start": first / rate, "end": last / rate, "gain": gain, "sources": written})
    return results
