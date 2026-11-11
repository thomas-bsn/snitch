"""Separate the voices of an audio window with a separation model this Worker holds ready (#401, ADR 0065).

This is the one entry point of a caller: audio in, the sources of the model out. It is independent of how the model
was installed and proven (``model_orders``, ``voice_models``), and nothing in the analysis of a clip calls it:
being ready opens no production job.

Nothing is ever fetched or installed here. A model that is absent, was never proven, or whose engine is gone is an
explicit refusal with a stable reason (:class:`VoiceProviderUnavailable`), never a silent fallback. The inference
runs in the environment of the engine bound to the model, through the fixed bridge. Standard library only.
"""
from __future__ import annotations

import math
import os
from collections.abc import Mapping
from typing import Any

from snitch_worker import engine_runtime, voice_models, voice_runtime
from snitch_worker.voice import VoiceProviderUnavailable

KIND = "separation"


def _seconds(value: float | None) -> float | None:
    if value is None:
        return None
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or value < 0:
        raise VoiceProviderUnavailable("inference_failed")
    return float(value)


def separate(environ: Mapping[str, str], repo: str, revision: str, audio: str, output_dir: str,
             start: float | None = None, end: float | None = None) -> dict[str, Any]:
    """Separate ``audio`` between ``start`` and ``end`` (seconds in the file, the whole file by default) with the
    separation model ``repo`` at ``revision``, and write one mono WAV per separated voice in ``output_dir``.

    Returns ``{"sample_rate", "start", "end", "gain", "sources": [{"index", "path", "samples"}, ...]}``: the rate is
    the model's, the window is the one really separated, and there are always as many sources as the model
    separates voices, whatever the audio holds. The sources keep their relative levels (one common ``gain``).

    Raises ``VoiceProviderUnavailable`` with ``model_path_missing`` (the model is absent or was never proven on
    this Worker), ``ml_libraries_missing`` (its engine is not installed, or it is not a separation model),
    ``gpu_out_of_memory``, ``model_load_failed`` or ``inference_failed``."""
    start, end = _seconds(start), _seconds(end)
    if start is not None and end is not None and end <= start:
        raise VoiceProviderUnavailable("inference_failed")
    path = voice_models.ready_path(environ, repo, revision, KIND) if isinstance(repo, str) and isinstance(revision, str) else ""
    if not path:
        raise VoiceProviderUnavailable("model_path_missing")
    try:
        manifest = engine_runtime.model_binding(path)
    except engine_runtime.RuntimeFailed:
        raise VoiceProviderUnavailable("model_load_failed") from None
    if manifest is None or not engine_runtime.is_on_demand(manifest) or manifest.get("kind") != KIND or not engine_runtime.installed(manifest, environ):
        raise VoiceProviderUnavailable("ml_libraries_missing")
    output_dir = os.path.abspath(output_dir)
    result = voice_runtime.call(manifest, path, environ, operation="infer", audio=audio, output_dir=output_dir, start=start, end=end)
    try:
        sources = result["sources"]
        valid = (type(result["sample_rate"]) is int and result["sample_rate"] > 0 and isinstance(sources, list) and len(sources) >= 2
                 and all(isinstance(source, dict) and source["index"] == index and type(source["samples"]) is int
                         and isinstance(source["path"], str) and os.path.dirname(os.path.abspath(source["path"])) == output_dir
                         for index, source in enumerate(sources)))
    except (KeyError, TypeError):
        valid = False
    if not valid:
        raise VoiceProviderUnavailable("inference_failed")
    return result
