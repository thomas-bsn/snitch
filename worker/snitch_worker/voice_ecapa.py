"""SpeechBrain ECAPA-TDNN fingerprints (engine ``speechbrain-ecapa``), loaded without executing model code.

SpeechBrain normally builds a model from ``hyperparams.yaml`` with HyperPyYAML, whose tags (``!new:``,
``!apply:``, ``!ref`` expressions) instantiate or call arbitrary Python objects. This adapter never runs that
program. Before anything is loaded it:

1. checks every file of the model directory against the SHA-256 pinned for a supported variant (a variant is a
   Hugging Face commit; an unknown commit is refused, not guessed);
2. parses ``hyperparams.yaml`` as data only (``yaml.compose``, nothing is constructed) and refuses every tag that
   is not a plain YAML value, a ``!ref`` of plain names, or a ``speechbrain.*`` / ``torch.*`` object outside a
   denylist of code-running modules;
3. reads the architecture from that file and refuses anything but the one supported ECAPA-TDNN;
4. scans the checkpoint pickle opcodes and only then calls ``torch.load(weights_only=True)``.

The architecture itself is built here, by this code, from constants. Every refusal raises
:class:`UnsupportedVariant` (stable reason ``model_load_failed``, a more precise ``detail`` that is safe to log).
"""
from __future__ import annotations

import hashlib
import math
import os
import pickletools
import re
import zipfile
from pathlib import Path
from typing import Any, Callable, Mapping

from snitch_worker.voice import VoiceProviderUnavailable
from snitch_worker.voice_pyannote import DecodedAudio

ENGINE = "speechbrain-ecapa"
ARCHITECTURE = "ecapa-tdnn"
SAMPLE_RATE = 16000
EMBEDDING_DIMENSIONS = 192

# Supported audio: explicit and small. Anything else is refused with a stable detail.
SUPPORTED_EXTENSIONS = (".wav", ".flac", ".ogg")
SUPPORTED_SAMPLE_RATES = (8000, 16000, 22050, 24000, 32000, 44100, 48000)
MAX_CHANNELS = 8
MAX_FILE_BYTES = 2 << 30
MAX_WINDOW_SECONDS = 120.0  # a longer window is embedded on its first 120 seconds (documented provenance)

MAX_YAML_BYTES = 64 << 10
MAX_CHECKPOINT_BYTES = 256 << 20

# Pinned variants: the files of a Hugging Face commit, by SHA-256. Only the architecture below is built.
VARIANTS: dict[str, dict[str, Any]] = {
    "speechbrain/spkrec-ecapa-voxceleb@0f99f2d0ebe89ac095bcc5903c4dd8f72b367286": {
        "files": {
            "hyperparams.yaml": "6f78854fa04ba59e761437b76a2575d3aba5e5016de3e9b69f0c9a5077fb1a41",
            "embedding_model.ckpt": "0575cb64845e6b9a10db9bcb74d5ac32b326b8dc90352671d345e2ee3d0126a2",
        },
        # Present in the commit, never loaded; hashed when found so that a substituted copy is refused too.
        "optional": {
            "classifier.ckpt": "fd9e3634fe68bd0a427c95e354c0c677374f62b3f434e45b78599950d860d535",
            "label_encoder.txt": "e13c3a167bb4112685670ee896d20e2b565af16b3a4ceeaa8689fa4d22adb8b9",
            "mean_var_norm_emb.ckpt": "cd70225b05b37be64fc5a95e24395d804231d43f74b2e1e5a513db7b69b34c33",
        },
    },
}
IGNORED_FILES = {"README.md", "config.json", "example1.wav", "example2.flac", ".gitattributes", ".snitch-ready", ".snitch-runtime.json"}

ARCH_CLASS = "speechbrain.lobes.models.ECAPA_TDNN.ECAPA_TDNN"
ARCH_PARAMS: dict[str, Any] = {
    "input_size": 80, "channels": [1024, 1024, 1024, 1024, 3072], "kernel_sizes": [5, 3, 3, 3, 1],
    "dilations": [1, 2, 3, 4, 1], "attention_channels": 128, "lin_neurons": EMBEDDING_DIMENSIONS,
}
FEATURES_CLASS = "speechbrain.lobes.features.Fbank"
NORM_CLASS = "speechbrain.processing.features.InputNormalization"

_STANDARD_TAGS = {"tag:yaml.org,2002:" + name for name in ("str", "int", "float", "bool", "null", "seq", "map")}
_OBJECT_TAGS = ("!new:", "!name:", "!apply:")
_DOTTED = re.compile(r"^(speechbrain|torch)(\.[A-Za-z][A-Za-z0-9_]*)+$")
_REF_NAME = re.compile(r"<([A-Za-z][A-Za-z0-9_]*)>")
_REF_REST = re.compile(r"^[A-Za-z0-9_/ .+*-]*$")
# Modules of the allowed namespaces that run code, read or write files, or reach the network.
_DENIED_PREFIXES = (
    "torch.load", "torch.save", "torch.hub", "torch.jit", "torch.package", "torch.serialization", "torch.distributed",
    "torch.multiprocessing", "torch.utils.cpp_extension", "torch.utils.bottleneck", "torch.utils.benchmark",
    "torch.compile", "torch.fx", "torch.onnx", "torch.export", "torch.library", "torch.testing", "torch._",
    "speechbrain.utils.superpowers", "speechbrain.utils.fetching", "speechbrain.utils.run_opts", "speechbrain.inference",
    "speechbrain.pretrained", "speechbrain.utils.hparams", "speechbrain.utils.logger", "speechbrain.utils.train_logger",
    "speechbrain.utils.checkpoints", "speechbrain.utils.distributed", "speechbrain.utils.data_pipeline",
    "speechbrain.dataio.dataio", "speechbrain.dataio.preprocess", "speechbrain.utils.edit_distance",
    "speechbrain.utils.profiling", "speechbrain.utils.autocast", "speechbrain.utils.importutils",
)
_ALLOWED_PICKLE_GLOBALS = {
    ("collections", "OrderedDict"), ("torch._utils", "_rebuild_tensor_v2"), ("torch._utils", "_rebuild_parameter"),
    *(("torch", name + "Storage") for name in ("Float", "Half", "BFloat16", "Double", "Long", "Int", "Short", "Byte", "Bool")),
}
_PICKLE_STRINGS = {"SHORT_BINUNICODE", "BINUNICODE", "BINUNICODE8", "UNICODE", "SHORT_BINSTRING", "BINSTRING", "STRING"}


class UnsupportedVariant(VoiceProviderUnavailable):
    """A model or audio this adapter does not support. ``detail`` is a stable code, never a path or library text."""

    def __init__(self, detail: str) -> None:
        super().__init__("model_load_failed")
        self.detail = detail


class UnsupportedAudio(VoiceProviderUnavailable):
    def __init__(self, detail: str) -> None:
        super().__init__("inference_failed")
        self.detail = detail


# -- files ---------------------------------------------------------------------------------------------------

def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        while chunk := stream.read(1 << 20):
            digest.update(chunk)
    return digest.hexdigest()


def verify_files(directory: str | os.PathLike[str], variants: Mapping[str, Mapping[str, Any]] | None = None) -> str:
    """The pinned variant this directory is, after checking every file against its SHA-256. Nothing is loaded."""
    variants = VARIANTS if variants is None else variants
    root = Path(directory)
    try:
        entries = {entry.name: entry for entry in root.iterdir()}
    except OSError:
        raise UnsupportedVariant("model_directory_unreadable") from None
    for name, entry in entries.items():
        if name == ".cache":
            continue  # huggingface_hub's download bookkeeping: never read
        if entry.is_symlink() or not entry.is_file():
            raise UnsupportedVariant("unexpected_directory_or_link")
        if name.endswith((".py", ".pyc", ".so", ".dll", ".sh")):
            raise UnsupportedVariant("code_file_in_model")  # no Hugging Face script, ever
    required_names = {n for v in variants.values() for n in v["files"]}
    if not required_names <= set(entries):
        raise UnsupportedVariant("model_file_missing")
    for key, variant in variants.items():
        wanted = {**variant.get("optional", {}), **variant["files"]}
        if not all(name in entries and _sha256(entries[name]) == digest for name, digest in variant["files"].items()):
            continue
        for name, digest in wanted.items():
            if name in entries and _sha256(entries[name]) != digest:
                raise UnsupportedVariant("file_hash_mismatch")
        extra = set(entries) - set(wanted) - IGNORED_FILES - {".cache"}
        if extra:
            raise UnsupportedVariant("unexpected_file")
        return key
    raise UnsupportedVariant("unsupported_revision")


# -- hyperparams.yaml, as data ---------------------------------------------------------------------------------

def _yaml() -> Any:
    try:
        import yaml  # noqa: PLC0415
    except ImportError:
        raise UnsupportedVariant("yaml_unavailable") from None
    return yaml


def _object_path(tag: str) -> str | None:
    for prefix in _OBJECT_TAGS:
        if tag.startswith(prefix):
            return tag[len(prefix):]
    return None


def check_object_path(dotted: str) -> None:
    if not _DOTTED.fullmatch(dotted):
        raise UnsupportedVariant("hyperparams_tag_refused")
    if any(dotted == p or dotted.startswith(p + ".") or (p.endswith("_") and dotted.startswith(p)) for p in _DENIED_PREFIXES):
        raise UnsupportedVariant("hyperparams_tag_refused")


def scan_hyperparams(text: str) -> Any:
    """Parse ``hyperparams.yaml`` without constructing anything and refuse every tag outside the allowlist.

    Returns the root mapping node. ``!!python/...``, ``!apply:os.system``, ``!new:subprocess.Popen``, ``!include``,
    ``!cmd``, a ``!ref`` that is an expression, or a ``!ref`` to a key that does not exist are all refused."""
    yaml = _yaml()
    if len(text.encode("utf-8", "replace")) > MAX_YAML_BYTES:
        raise UnsupportedVariant("hyperparams_too_large")
    try:
        root = yaml.compose(text, Loader=yaml.SafeLoader)
    except yaml.YAMLError:
        raise UnsupportedVariant("hyperparams_invalid") from None
    if not isinstance(root, yaml.MappingNode):
        raise UnsupportedVariant("hyperparams_invalid")
    keys = {n.value for n, _ in root.value if isinstance(n, yaml.ScalarNode)}
    seen: set[int] = set()

    def walk(node: Any, depth: int = 0) -> None:
        if depth > 32 or len(seen) > 5000:
            raise UnsupportedVariant("hyperparams_invalid")
        if id(node) in seen:
            return
        seen.add(id(node))
        tag = node.tag
        if tag == "!ref":
            if not isinstance(node, yaml.ScalarNode):
                raise UnsupportedVariant("hyperparams_tag_refused")
            names = _REF_NAME.findall(node.value)
            rest = _REF_NAME.sub("", node.value)
            if not names or "__" in rest or not _REF_REST.fullmatch(rest) or any(n not in keys for n in names):
                raise UnsupportedVariant("hyperparams_tag_refused")
        elif (dotted := _object_path(tag)) is not None:
            check_object_path(dotted)
        elif tag not in _STANDARD_TAGS:
            raise UnsupportedVariant("hyperparams_tag_refused")
        if isinstance(node, yaml.MappingNode):
            for key, value in node.value:
                walk(key, depth + 1)
                walk(value, depth + 1)
        elif isinstance(node, yaml.SequenceNode):
            for item in node.value:
                walk(item, depth + 1)

    walk(root)
    return root


def _plain(node: Any, scalars: Mapping[str, Any]) -> Any:
    """The value of a plain node (or a ``!ref <name>`` of a plain top-level scalar). Never an object."""
    yaml = _yaml()
    if node.tag == "!ref" and isinstance(node, yaml.ScalarNode):
        match = _REF_NAME.fullmatch(node.value)
        if match and match.group(1) in scalars:
            return scalars[match.group(1)]
        raise UnsupportedVariant("unsupported_architecture")
    if isinstance(node, yaml.ScalarNode) and node.tag in _STANDARD_TAGS:
        return yaml.SafeLoader("").construct_object(node)
    if isinstance(node, yaml.SequenceNode) and node.tag in _STANDARD_TAGS:
        return [_plain(item, scalars) for item in node.value]
    raise UnsupportedVariant("unsupported_architecture")


def check_architecture(root: Any) -> None:
    """The file must describe exactly the ECAPA-TDNN this adapter builds; any other variant is refused."""
    yaml = _yaml()
    top = {key.value: value for key, value in root.value if isinstance(key, yaml.ScalarNode)}
    scalars = {name: _plain(node, {}) for name, node in top.items()
               if isinstance(node, yaml.ScalarNode) and node.tag in _STANDARD_TAGS}

    def kwargs(name: str, cls: str) -> dict[str, Any]:
        node = top.get(name)
        if node is None or node.tag != "!new:" + cls or not isinstance(node, yaml.MappingNode):
            raise UnsupportedVariant("unsupported_architecture")
        return {key.value: _plain(value, scalars) for key, value in node.value}

    if kwargs("embedding_model", ARCH_CLASS) != ARCH_PARAMS:
        raise UnsupportedVariant("unsupported_architecture")
    if kwargs("compute_features", FEATURES_CLASS) != {"n_mels": ARCH_PARAMS["input_size"]}:
        raise UnsupportedVariant("unsupported_features")
    if kwargs("mean_var_norm", NORM_CLASS) != {"norm_type": "sentence", "std_norm": False}:
        raise UnsupportedVariant("unsupported_features")


# -- checkpoint ----------------------------------------------------------------------------------------------------

def scan_checkpoint(path: str | os.PathLike[str]) -> None:
    """Refuse a checkpoint whose pickle references anything but tensors and an ordered dict (no code object)."""
    target = Path(path)
    try:
        if target.stat().st_size > MAX_CHECKPOINT_BYTES:
            raise UnsupportedVariant("checkpoint_too_large")
        with zipfile.ZipFile(target) as archive:
            pickles = [n for n in archive.namelist() if n.endswith("/data.pkl") or n == "data.pkl"]
            if len(pickles) != 1:
                raise UnsupportedVariant("checkpoint_unsupported_format")
            info = archive.getinfo(pickles[0])
            if info.file_size > 32 << 20:
                raise UnsupportedVariant("checkpoint_unsupported_format")
            data = archive.read(pickles[0])
    except zipfile.BadZipFile:
        raise UnsupportedVariant("checkpoint_unsupported_format") from None  # legacy pickle: never read
    except OSError:
        raise UnsupportedVariant("model_directory_unreadable") from None
    strings: list[str] = []
    try:
        for opcode, argument, _ in pickletools.genops(data):
            if opcode.name in _PICKLE_STRINGS:
                strings.append(argument if isinstance(argument, str) else argument.decode("utf-8", "replace"))
            elif opcode.name == "GLOBAL":
                module, _, name = str(argument).partition(" ")
                if (module, name) not in _ALLOWED_PICKLE_GLOBALS:
                    raise UnsupportedVariant("checkpoint_forbidden_object")
            elif opcode.name == "STACK_GLOBAL":
                if len(strings) < 2 or (strings[-2], strings[-1]) not in _ALLOWED_PICKLE_GLOBALS:
                    raise UnsupportedVariant("checkpoint_forbidden_object")
            elif opcode.name in {"INST", "OBJ", "EXT1", "EXT2", "EXT4", "BYTEARRAY8", "NEXT_BUFFER", "READONLY_BUFFER"}:
                raise UnsupportedVariant("checkpoint_forbidden_object")
    except UnsupportedVariant:
        raise
    except Exception:  # noqa: BLE001 - a malformed stream is a refusal, its text never travels
        raise UnsupportedVariant("checkpoint_unsupported_format") from None


# -- the model -----------------------------------------------------------------------------------------------------

def load_embedder(directory: str, device: str, variants: Mapping[str, Mapping[str, Any]] | None = None) -> Callable[[Any], Any]:
    """Verify, scan, then build the pinned architecture and load its weights. Returns ``embed(waveform) -> tensor``.

    ``waveform`` is a ``[1, samples]`` float tensor at 16 kHz. Offline: nothing is fetched, no hub, no from_hparams."""
    verify_files(directory, variants)
    root_dir = Path(directory)
    check_architecture(scan_hyperparams((root_dir / "hyperparams.yaml").read_text(encoding="utf-8")))
    scan_checkpoint(root_dir / "embedding_model.ckpt")
    try:
        import torch  # noqa: PLC0415 - lazy: the Worker runs without the ML stack
        from speechbrain.lobes.features import Fbank  # noqa: PLC0415
        from speechbrain.lobes.models.ECAPA_TDNN import ECAPA_TDNN  # noqa: PLC0415
        from speechbrain.processing.features import InputNormalization  # noqa: PLC0415
    except ImportError:
        raise VoiceProviderUnavailable("ml_libraries_missing") from None
    model = ECAPA_TDNN(**ARCH_PARAMS).to(device).eval()
    state = torch.load(root_dir / "embedding_model.ckpt", map_location=device, weights_only=True)
    model.load_state_dict(state, strict=True)
    features = Fbank(n_mels=ARCH_PARAMS["input_size"]).to(device).eval()
    norm = InputNormalization(norm_type="sentence", std_norm=False).to(device)

    def embed(waveform: Any) -> Any:
        signal = features(waveform)
        return model(norm(signal, torch.ones(signal.shape[0], device=device)))

    return embed


def check_audio_file(path: str) -> None:
    """Refuse an audio file that is not one of the supported containers or is absurdly large."""
    if not path.lower().endswith(SUPPORTED_EXTENSIONS):
        raise UnsupportedAudio("unsupported_audio_format")
    try:
        if os.path.getsize(path) > MAX_FILE_BYTES:
            raise UnsupportedAudio("audio_too_large")
    except OSError:
        raise UnsupportedAudio("audio_unreadable") from None


def speaker_loader(directory: str, device: str, variants: Mapping[str, Mapping[str, Any]] | None = None) -> Callable[[str], DecodedAudio]:
    """Same shape as the other embedders: ``open_audio(path)`` returns a :class:`DecodedAudio`."""
    import torch  # noqa: PLC0415
    import torchaudio  # noqa: PLC0415

    embed = load_embedder(directory, device, variants)

    def open_audio(audio_path: str) -> DecodedAudio:
        check_audio_file(audio_path)
        try:
            waveform, rate = torchaudio.load(audio_path)
        except Exception:  # noqa: BLE001 - library text never travels
            raise UnsupportedAudio("audio_undecodable") from None
        channels = int(waveform.shape[0])
        if not 1 <= channels <= MAX_CHANNELS:
            raise UnsupportedAudio("unsupported_channels")
        if rate not in SUPPORTED_SAMPLE_RATES:
            raise UnsupportedAudio("unsupported_sample_rate")
        if not bool(torch.isfinite(waveform).all()):
            raise UnsupportedAudio("audio_not_finite")
        waveform = waveform.mean(dim=0, keepdim=True)
        if rate != SAMPLE_RATE:
            waveform = torchaudio.functional.resample(waveform, rate, SAMPLE_RATE)
        waveform = waveform.to(device)

        def window(start: float, end: float) -> list[float]:
            end = min(end, start + MAX_WINDOW_SECONDS)
            chunk = waveform[:, int(start * SAMPLE_RATE):int(end * SAMPLE_RATE)]
            with torch.inference_mode():
                vector = embed(chunk).detach().cpu().reshape(-1).tolist()
            if len(vector) != EMBEDDING_DIMENSIONS or not all(math.isfinite(x) for x in vector):
                raise VoiceProviderUnavailable("inference_failed")
            return vector

        return DecodedAudio(window, SAMPLE_RATE, channels, int(waveform.shape[-1]))

    return open_audio
