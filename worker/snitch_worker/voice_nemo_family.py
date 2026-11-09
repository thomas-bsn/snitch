"""Generic adapter of the NeMo family for on-demand engines (#373, ADR 0056).

One adapter per library, not per model: any NeMo diarization model (Nemotron-3, Sortformer...) and TitaNet
fingerprints run through the Python of the on-demand environment, called by ``engine_bridge``. The safeguards
are fixed here and no model, order or release can switch them off:

* no code of the model repository runs: a repository holding Python files, or asking for remote code
  (``auto_map``, ``trust_remote_code``), is refused before anything is imported;
* the configuration of the ``.nemo`` archive is read as data (``yaml.safe_load``): every class it names must
  belong to NeMo or ``torch.nn``, interpolations are plain node references (no resolver such as ``oc.env``),
  and no object can be built from it;
* weights are read in safe mode only: ``torch.load(weights_only=True)`` proves the checkpoint holds tensors
  before NeMo opens the archive (a pickle that would execute code is refused);
* no Hugging Face token reaches this process (the Worker scrubs the environment, offline mode is forced).

A refusal is a ``model_load_failed`` reason with a stable ``detail`` code, which the Worker log carries and
the model verification folds onto its existing ``runtime_incompatible`` cause. Fingerprints reuse the fixed,
byte-checked TitaNet loader of ``voice_titanet``.
"""

from __future__ import annotations

import io
import json
import os
import re
from pathlib import Path
from typing import Any, Callable, Mapping

from snitch_worker import voice_titanet
from snitch_worker.voice import VoiceProviderUnavailable

ALLOWED_TARGETS = ("nemo.", "torch.nn.")
_REFERENCE = re.compile(r"\$\{[A-Za-z0-9_.]+\}")
_REMOTE_CODE_KEYS = {"auto_map", "trust_remote_code"}


class FamilyRefused(voice_titanet.TitaNetRefused):
    """The model is refused by a fixed safeguard. ``reason`` stays ``model_load_failed``; ``detail`` is the stable code."""


def _contains_remote_code_request(value: Any, depth: int = 0) -> bool:
    if depth > 10:
        return True
    if isinstance(value, dict):
        return any(key in _REMOTE_CODE_KEYS or _contains_remote_code_request(item, depth + 1) for key, item in value.items())
    if isinstance(value, list):
        return any(_contains_remote_code_request(item, depth + 1) for item in value)
    return False


def check_no_remote_code(path: str) -> None:
    """Nothing of the repository is imported or executed: its code files and its requests for code are refusals."""
    if not os.path.isdir(path):
        return
    try:
        for root, _, names in os.walk(path):
            if any(name.endswith(".py") for name in names):
                raise FamilyRefused("remote_code_required")
        config = Path(path, "config.json")
        if config.is_file() and config.stat().st_size <= voice_titanet.MAX_CONFIG_BYTES:
            try:
                data = json.loads(config.read_text())
            except ValueError:
                raise FamilyRefused("config_unreadable") from None
            if _contains_remote_code_request(data):
                raise FamilyRefused("remote_code_required")
    except OSError:
        raise FamilyRefused("archive_missing") from None


def _walk(value: Any, depth: int, top: bool) -> None:
    if depth > 30:
        raise FamilyRefused("config_executable")
    if isinstance(value, dict):
        for key, item in value.items():
            if not isinstance(key, str):
                raise FamilyRefused("config_executable")
            if key == "_target_" or (key == "target" and top):
                if not isinstance(item, str) or not item.startswith(ALLOWED_TARGETS):
                    raise FamilyRefused("config_executable")
                continue
            if key.startswith("_") or key == "target" or key == "cls":
                raise FamilyRefused("config_executable")
            _walk(item, depth + 1, False)
    elif isinstance(value, list):
        for item in value:
            _walk(item, depth + 1, False)
    elif isinstance(value, str):
        # Interpolation is a reference to another node, never a resolver (oc.env, oc.decode...).
        remaining = _REFERENCE.sub("", value)
        if "${" in remaining or "${oc." in value:
            raise FamilyRefused("config_executable")
    elif value is not None and not isinstance(value, (int, float, bool)):
        raise FamilyRefused("config_executable")


def check_config(raw: bytes) -> dict[str, Any]:
    """The archive configuration as plain data, or a refusal."""
    try:
        config = voice_titanet.parse_config(raw)
    except voice_titanet.TitaNetRefused as refused:
        raise FamilyRefused(refused.detail) from None
    _walk(config, 0, True)
    return config


def check_weights(raw: bytes) -> None:
    """Prove the checkpoint holds tensors only: the safe unpickler refuses anything that would run code."""
    import torch  # noqa: PLC0415 - only in the on-demand environment

    try:
        torch.load(io.BytesIO(raw), map_location="cpu", weights_only=True)
    except Exception:  # noqa: BLE001 - the library message may quote the payload
        raise FamilyRefused("weights_unsafe") from None


def preflight(path: str, load_weights: Callable[[bytes], None] | None = None) -> str:
    """Every fixed safeguard, before NeMo sees the model. Returns the archive to restore."""
    check_no_remote_code(path)
    try:
        config, weights = voice_titanet.read_archive(path)
    except voice_titanet.TitaNetRefused as refused:
        raise FamilyRefused(refused.detail) from None
    check_config(config)
    (load_weights or check_weights)(weights)
    return voice_titanet.archive_of(path)


def _restore(archive: str, device: str) -> Any:  # pragma: no cover - needs NeMo and the model (#377)
    import torch
    from nemo.core import ModelPT

    model = ModelPT.restore_from(restore_path=archive, map_location=torch.device(device))
    model.eval()
    return model


def load_diarizer(path: str, device: str, restore: Callable[[str, str], Any] | None = None,
                  load_weights: Callable[[bytes], None] | None = None) -> Callable[[str], Any]:
    archive = preflight(path, load_weights)
    model = (restore or _restore)(archive, device)

    def run(audio_path: str) -> Any:
        return model.diarize(audio=[audio_path], batch_size=1)[0]

    return run


def provider(runtime: Mapping[str, Any], environ: Any, **injected: Any) -> Any:
    """The provider of an on-demand NeMo runtime, chosen by its kind. ``injected`` is for tests only."""
    from snitch_worker.engine_bridge import SpeakerProvider
    from snitch_worker.voice_nemo import NemoProvider

    kind = runtime.get("kind")
    if kind == "fingerprint":
        loader = injected.pop("load_embedder", None)
        built = SpeakerProvider("nemo-titanet", environ=environ, **injected)
        if loader is not None:
            built._load_embedder = loader
        return built
    if kind == "diarization":
        return NemoProvider(environ=environ, load_pipeline=injected.pop("load_pipeline", load_diarizer), **injected)
    raise VoiceProviderUnavailable("model_load_failed")
