"""NVIDIA Sortformer diarization (NeMo): same speakers and intervals as the pyannote adapter.

Only the diarization step differs; fingerprints reuse the pyannote embedder (the catalogue has a
single embedding model). The model is read offline from the active model's directory, which holds
one ``.nemo`` archive. NeMo is imported lazily inside the loader: without it the provider raises
``ml_libraries_missing`` and the model preparation reports ``runtime_incompatible``.
"""

from __future__ import annotations

import glob
import os
from typing import Any, Callable, Iterable

from snitch_worker import cancellation
from snitch_worker.voice import VoiceProviderUnavailable, diarization_from_turns
from snitch_worker.voice_pyannote import PyannoteProvider


def nemo_archive(path: str) -> str:
    """The ``.nemo`` archive of a model directory (or the file itself), else an empty string."""
    if os.path.isfile(path):
        return path if path.endswith(".nemo") else ""
    found = sorted(glob.glob(os.path.join(path, "*.nemo")))
    return found[0] if found else ""


def parse_segments(segments: Iterable[Any]) -> list[tuple[float, float, str]]:
    """``"<begin> <end> <speaker>"`` strings (NeMo's output) or ``(begin, end, speaker)`` tuples."""
    turns: list[tuple[float, float, str]] = []
    for item in segments:
        if isinstance(item, str):
            begin, end, label = item.split()[:3]
        else:
            begin, end, label = item
        turns.append((float(begin), float(end), str(label)))
    return turns


def _load_sortformer(path: str, device: str) -> Callable[[str], Any]:  # pragma: no cover - needs NeMo and the model
    import torch
    from nemo.collections.asr.models import SortformerEncLabelModel

    archive = nemo_archive(path)
    if not archive:
        raise VoiceProviderUnavailable("model_path_missing")
    model = SortformerEncLabelModel.restore_from(restore_path=archive, map_location=torch.device(device))
    model.eval()

    def run(audio_path: str) -> Any:
        return model.diarize(audio=[audio_path], batch_size=1)[0]

    return run


class NemoProvider(PyannoteProvider):
    def __init__(self, *args: Any, load_pipeline: Callable[[str, str], Callable[[str], Any]] | None = None, **kwargs: Any) -> None:
        super().__init__(*args, load_pipeline=load_pipeline or _load_sortformer, **kwargs)

    def diarize(self, config: dict[str, Any], audio_path: str, stream_index: int) -> dict[str, Any]:
        run = self._pipeline_loaded(config)
        try:
            with self._inference_lock:
                # Sortformer offers no progress hook: a refused lease interrupts it (ADR 0029).
                turns = parse_segments(cancellation.interruptible(run, audio_path))
        except VoiceProviderUnavailable:
            raise
        except Exception:  # noqa: BLE001 - library messages may carry paths
            raise VoiceProviderUnavailable("inference_failed") from None
        return diarization_from_turns(config, turns, stream_index)
