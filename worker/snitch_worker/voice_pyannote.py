"""Real voice provider: pyannote diarization and speaker embeddings.

Design constraints (see docs/real-voice-worker.md):

* torch and pyannote are imported lazily, inside the loaders only: the worker and
  its tests start without any ML library.
* No implicit download. Models are read from local paths given by environment
  variables, and Hugging Face offline mode is forced before any library loads. The
  Hugging Face token is never read here; downloads belong to the explicit, opt-in
  ``snitch_worker.bootstrap`` (run before the worker) or ``worker/scripts/download_models.py``.
* Models are loaded once per process and inference is serialised (shared VRAM).
* Every failure is raised as ``VoiceProviderUnavailable`` with a stable reason code
  and no library message, path or token (the cause is suppressed).
"""

from __future__ import annotations

import logging
import os
import threading
import time
from typing import Any, Callable, MutableMapping

from snitch_worker import log, voice_manifest, voice_models
from snitch_worker.voice import (
    VoiceProviderUnavailable,
    diarization_from_turns,
    fingerprints_from_embedder,
)

DIARIZATION_PATH_ENV = "SNITCH_VOICE_DIARIZATION_MODEL_PATH"
EMBEDDING_PATH_ENV = "SNITCH_VOICE_EMBEDDING_MODEL_PATH"
EMBEDDING_NAME_ENV = "SNITCH_VOICE_EMBEDDING_MODEL_NAME"
EMBEDDING_VERSION_ENV = "SNITCH_VOICE_EMBEDDING_MODEL_VERSION"
DEVICE_ENV = "SNITCH_VOICE_DEVICE"
DEFAULT_EMBEDDING_NAME = "pyannote/wespeaker-voxceleb-resnet34-LM"
# Options a diarization trial may impose on the pipeline; a regular job never carries them.
SPEAKER_OPTIONS = ("num_speakers", "min_speakers", "max_speakers")

PipelineLoader = Callable[[str, str], Callable[[str], Any]]
EmbedderLoader = Callable[[str, str], Callable[[str], Callable[[float, float], Any]]]


def _classify(exc: BaseException, default: str) -> str:
    """A stable reason for a library failure, from its class and text, which are never kept or logged."""
    kind = type(exc).__name__.lower()
    text = str(exc).lower()
    if "outofmemory" in kind or "out of memory" in text:
        return "gpu_out_of_memory"
    if isinstance(exc, PermissionError) or "gatedrepo" in kind or "unauthorized" in kind or "forbidden" in kind or " 401" in f" {text}" or " 403" in f" {text}":
        return "model_access_denied"
    return default


def _probe_device() -> str:
    try:
        import torch  # noqa: PLC0415 - lazy on purpose

        return "cuda" if torch.cuda.is_available() else "cpu"
    except Exception:  # noqa: BLE001
        raise VoiceProviderUnavailable("ml_libraries_missing") from None


def _load_pipeline(path: str, device: str) -> Callable[[str], Any]:  # pragma: no cover - needs models
    import torch
    from pyannote.audio import Pipeline

    config = os.path.join(path, "config.yaml") if os.path.isdir(path) else path
    pipeline = Pipeline.from_pretrained(config)
    pipeline.to(torch.device(device))
    return pipeline


def _load_embedder(path: str, device: str) -> Callable[[str], Callable[[float, float], Any]]:  # pragma: no cover - needs models
    import torch
    import torchaudio
    from pyannote.audio import Inference, Model
    from pyannote.core import Segment

    inference = Inference(Model.from_pretrained(path), window="whole", device=torch.device(device))

    def open_audio(audio_path: str) -> Callable[[float, float], Any]:
        waveform, sample_rate = torchaudio.load(audio_path)  # decoded once per task
        audio = {"waveform": waveform, "sample_rate": sample_rate}

        def embed(start: float, end: float) -> Any:
            return inference.crop(audio, Segment(start, end)).reshape(-1).tolist()

        return embed

    return open_audio


class PyannoteProvider:
    def __init__(
        self,
        environ: MutableMapping[str, str] | None = None,
        load_pipeline: PipelineLoader | None = None,
        load_embedder: EmbedderLoader | None = None,
        device_probe: Callable[[], str] | None = None,
    ) -> None:
        self._environ = os.environ if environ is None else environ
        self._load_pipeline = load_pipeline or _load_pipeline
        self._load_embedder = load_embedder or _load_embedder
        self._probe = device_probe or _probe_device
        self._load_lock = threading.Lock()
        self._inference_lock = threading.Lock()
        self._pipeline: Callable[[str], Any] | None = None
        self._open_audio: Callable[[str], Callable[[float, float], Any]] | None = None
        self._device: str | None = None
        self._pipeline_path: str | None = None

    # -- configuration -------------------------------------------------

    def _path(self, variable: str) -> str:
        path = self._environ.get(variable, "").strip()
        if not path or not os.path.exists(path):
            raise VoiceProviderUnavailable("model_path_missing")
        return path

    def _diarization_path(self, config: dict[str, Any]) -> str:
        """The active model's own directory on the models volume, else the legacy single path."""
        repo, revision = str(config.get("model") or ""), str(config.get("model_version") or "")
        if voice_models.is_ready(self._environ, repo, revision):
            return voice_models.model_path(self._environ, repo, revision)
        return self._path(DIARIZATION_PATH_ENV)

    def _embedding_identity(self) -> tuple[str, str]:
        version = self._environ.get(EMBEDDING_VERSION_ENV, "").strip()
        if not version:  # fall back to the marker written by snitch_worker.bootstrap
            version = voice_manifest.read_embedding_version(self._environ)
        if not version:
            raise VoiceProviderUnavailable("embedding_version_missing")
        return self._environ.get(EMBEDDING_NAME_ENV, "").strip() or DEFAULT_EMBEDDING_NAME, version

    def ensure_ready(self, config: dict[str, Any]) -> None:
        """Cheap checks only: nothing is imported, loaded or downloaded."""
        self._diarization_path(config)
        self._path(EMBEDDING_PATH_ENV)
        self._embedding_identity()

    def ensure_diarization_ready(self, config: dict[str, Any]) -> None:
        """A trial only needs its own diarization model."""
        self._diarization_path(config)

    def _resolve_device(self) -> str:
        if self._device is None:
            wanted = self._environ.get(DEVICE_ENV, "auto").strip().lower() or "auto"
            if wanted not in {"auto", "cuda", "cpu"}:
                raise VoiceProviderUnavailable("device_invalid")
            if wanted == "cpu":
                self._device = "cpu"
            else:
                available = self._probe()
                if wanted == "cuda" and available != "cuda":
                    raise VoiceProviderUnavailable("cuda_unavailable")
                self._device = available  # auto: GPU when present, otherwise CPU
        return self._device

    def _offline(self) -> None:
        self._environ["HF_HUB_OFFLINE"] = "1"
        self._environ["TRANSFORMERS_OFFLINE"] = "1"

    # -- lazy, cached model loading -------------------------------------

    def _pipeline_loaded(self, config: dict[str, Any]) -> Callable[[str], Any]:
        with self._load_lock:
            path = self._diarization_path(config)
            if self._pipeline is None or self._pipeline_path != path:
                self._pipeline = None
                device = self._resolve_device()
                self._offline()
                started = time.monotonic()
                try:
                    self._pipeline = self._load_pipeline(path, device)
                    self._pipeline_path = path
                except VoiceProviderUnavailable:
                    raise
                except ImportError:
                    raise VoiceProviderUnavailable("ml_libraries_missing") from None
                except Exception as exc:  # noqa: BLE001 - library messages may carry paths or tokens
                    raise VoiceProviderUnavailable(_classify(exc, "model_load_failed")) from None
                log.emit(logging.INFO, "diarization_model_loaded", device=device, load_ms=int((time.monotonic() - started) * 1000))
            return self._pipeline

    def _audio_opener(self) -> Callable[[str], Callable[[float, float], Any]]:
        with self._load_lock:
            if self._open_audio is None:
                path = self._path(EMBEDDING_PATH_ENV)
                device = self._resolve_device()
                self._offline()
                try:
                    self._open_audio = self._load_embedder(path, device)
                except VoiceProviderUnavailable:
                    raise
                except ImportError:
                    raise VoiceProviderUnavailable("ml_libraries_missing") from None
                except Exception:  # noqa: BLE001
                    raise VoiceProviderUnavailable("model_load_failed") from None
            return self._open_audio

    # -- steps ------------------------------------------------------------

    def diarize(self, config: dict[str, Any], audio_path: str, stream_index: int) -> dict[str, Any]:
        pipeline = self._pipeline_loaded(config)
        options = {name: int(config[name]) for name in SPEAKER_OPTIONS if config.get(name)}
        started = time.monotonic()
        try:
            with self._inference_lock:
                output = pipeline(audio_path, **options)
            annotation = getattr(output, "speaker_diarization", output)
            turns = [(segment.start, segment.end, label) for segment, _track, label in annotation.itertracks(yield_label=True)]
        except VoiceProviderUnavailable:
            raise
        except Exception as exc:  # noqa: BLE001
            raise VoiceProviderUnavailable(_classify(exc, "inference_failed")) from None
        log.emit(logging.INFO, "diarization_inference_done", device=self._device, inference_ms=int((time.monotonic() - started) * 1000))
        return diarization_from_turns(config, turns, stream_index)

    def fingerprint(self, config: dict[str, Any], diarization: dict[str, Any], audio_path: str, stream_index: int) -> dict[str, Any]:
        name, version = self._embedding_identity()
        state: dict[str, Callable[[float, float], Any]] = {}

        def embed(start: float, end: float) -> Any:
            if "embed" not in state:  # models load only if a turn is long enough
                state["embed"] = self._audio_opener()(audio_path)
            with self._inference_lock:
                return state["embed"](start, end)

        try:
            return fingerprints_from_embedder(config, diarization, embed, stream_index, name, version)
        except VoiceProviderUnavailable:
            raise
        except Exception:  # noqa: BLE001
            raise VoiceProviderUnavailable("inference_failed") from None
