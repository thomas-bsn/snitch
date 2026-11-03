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
import re
import threading
import time
from typing import Any, Callable, MutableMapping

from snitch_worker import log, voice_manifest, voice_models
from snitch_worker.voice import (
    MIN_EMBEDDABLE_SECONDS,
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
EmbedderLoader = Callable[[str, str], Callable[[str], "DecodedAudio"]]


_STATUS_TEXT = re.compile(r"(?<![\w.-])(401|403)(?![\w.-])")
_STATUS_CONTEXT = ("http", "status", "client error", "unauthorized", "forbidden", "access")


def _http_status(exc: BaseException) -> int | None:
    for holder in (exc, getattr(exc, "response", None)):
        for name in ("status_code", "status"):
            value = getattr(holder, name, None)
            if isinstance(value, int) and not isinstance(value, bool):
                return value
    return None


def _classify(exc: BaseException, default: str) -> str:
    """A stable reason for a library failure: exception type and HTTP status first.

    The text is read, lowercased and never kept or logged, only as a last resort when neither the type
    nor a status says anything; a number is a status only next to an HTTP-looking word.
    """
    kind = type(exc).__name__.lower()
    if isinstance(exc, MemoryError) or "outofmemory" in kind:
        return "gpu_out_of_memory"
    if isinstance(exc, PermissionError) or "gatedrepo" in kind or "unauthorized" in kind or "forbidden" in kind:
        return "model_access_denied"
    status = _http_status(exc)
    if status is not None:
        return "model_access_denied" if status in (401, 403) else default
    text = str(exc).lower()
    if "out of memory" in text:
        return "gpu_out_of_memory"
    if _STATUS_TEXT.search(text) and any(word in text for word in _STATUS_CONTEXT):
        return "model_access_denied"
    return default


def audio_window(start: float, end: float, num_samples: int, sample_rate: int) -> tuple[float, float] | None:
    """The part of a turn that lies inside the audio, or ``None`` when it is too short to embed.

    pyannote crops an in-memory waveform only when the window ends strictly before the last sample
    (``round(end * rate) < num_samples``), otherwise it raises. The window therefore stops at the last
    valid sample, ``(num_samples - 1) / rate``. Sortformer and pyannote turns can end on, or a few
    hundredths of a second past, the end of the clip. Only the embedded window is cut: the diarization
    keeps the original timestamps.
    """
    low = max(0.0, float(start))
    high = min(float(end), (num_samples - 1) / sample_rate)
    if high - low < MIN_EMBEDDABLE_SECONDS:
        return None
    return low, high


class DecodedAudio:
    """A decoded audio track and the embedding of one of its windows.

    ``sample_rate``, ``channels`` and ``num_samples`` describe the decoded waveform: the provider keeps
    every window inside the audio with them and logs what it was working on.
    """

    def __init__(self, embed: Callable[[float, float], Any], sample_rate: int, channels: int, num_samples: int) -> None:
        self._embed = embed
        self.sample_rate, self.channels, self.num_samples = sample_rate, channels, num_samples

    def __call__(self, start: float, end: float) -> Any:
        return self._embed(start, end)


SUMMARY_LIMIT = 240
_SECRET_TEXT = (
    re.compile(r"hf_[A-Za-z0-9]{6,}"),
    re.compile(r"(?i)\b(bearer|token|authorization|api[_-]?key|password|secret)\b\s*[:=]?\s*(?:(?:bearer|basic)\s+)?\S+"),
    re.compile(r"[A-Za-z][A-Za-z0-9+.-]*://\S+"),
    re.compile(r"(?<![\w.:])(?:[A-Za-z]:)?(?:[\\/][^\\/\s'\"()]+)+[\\/]?"),
    re.compile(r"[A-Za-z0-9_+=-]{32,}"),
)


def error_summary(exc: BaseException) -> str:
    """A short excerpt of a library message, with tokens, URLs, paths and long opaque strings replaced (ADR 0028)."""
    text = " ".join(str(exc).split())
    for pattern in _SECRET_TEXT:
        text = pattern.sub("[redacted]", text)
    return text[:SUMMARY_LIMIT]


def _original(exc: BaseException) -> BaseException:
    """The library exception behind a ``VoiceProviderUnavailable`` raised ``from None``, kept as its hidden context."""
    if isinstance(exc, VoiceProviderUnavailable) and exc.__suppress_context__ and exc.__context__ is not None:
        return exc.__context__
    return exc


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


def _load_embedder(path: str, device: str) -> Callable[[str], DecodedAudio]:  # pragma: no cover - needs models
    import torch
    import torchaudio
    from pyannote.audio import Inference, Model
    from pyannote.core import Segment

    inference = Inference(Model.from_pretrained(path), window="whole", device=torch.device(device))

    def open_audio(audio_path: str) -> DecodedAudio:
        waveform, sample_rate = torchaudio.load(audio_path)  # decoded once per task
        audio = {"waveform": waveform, "sample_rate": sample_rate}

        def embed(start: float, end: float) -> Any:
            # pyannote downmixes and resamples the window to the model's 16 kHz mono itself.
            return inference.crop(audio, Segment(start, end)).reshape(-1).tolist()

        channels, num_samples = (int(n) for n in waveform.shape)
        return DecodedAudio(embed, int(sample_rate), channels, num_samples)

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
        self._open_audio: Callable[[str], DecodedAudio] | None = None
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

    def _audio_opener(self) -> Callable[[str], DecodedAudio]:
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
            reason = _classify(exc, "inference_failed")
            log.emit(logging.WARNING, "diarization_inference_failed", code=reason, exc_class=type(exc).__name__, device=self._device)
            raise VoiceProviderUnavailable(reason) from None
        log.emit(logging.INFO, "diarization_inference_done", device=self._device, inference_ms=int((time.monotonic() - started) * 1000))
        return diarization_from_turns(config, turns, stream_index)

    def fingerprint(self, config: dict[str, Any], diarization: dict[str, Any], audio_path: str, stream_index: int) -> dict[str, Any]:
        name, version = self._embedding_identity()
        state: dict[str, DecodedAudio] = {}
        # What the step was doing, for the diagnostic log: never audio content, path or token.
        trace: dict[str, Any] = {"stage": "segment_extraction", "embedded": 0, "skipped_out_of_audio": 0, "cut_at_audio_end": 0}

        def embed(start: float, end: float) -> Any:
            if "audio" not in state:  # models load only if a turn is long enough
                trace["stage"] = "model_load"
                opener = self._audio_opener()
                trace["stage"] = "audio_decode"
                audio = state["audio"] = opener(audio_path)
                trace.update(sample_rate=audio.sample_rate, channels=audio.channels, num_samples=audio.num_samples)
            audio = state["audio"]
            trace["stage"] = "segment_extraction"
            trace["segment_start"], trace["segment_end"] = round(start, 3), round(end, 3)
            window = audio_window(start, end, audio.num_samples, audio.sample_rate)
            if window is None:
                trace["skipped_out_of_audio"] += 1
                return None  # outside (or too short once cut at the end of) the audio: no embedding for this turn
            if window[1] < end:
                trace["cut_at_audio_end"] += 1
            trace["input_shape"] = [audio.channels, round((window[1] - window[0]) * audio.sample_rate)]
            trace["stage"] = "embedding_inference"
            with self._inference_lock:
                vector = audio(*window)
            trace["embedded"] += 1
            trace["stage"] = "post_processing"
            return vector

        started = time.monotonic()
        try:
            result = fingerprints_from_embedder(config, diarization, embed, stream_index, name, version)
        except VoiceProviderUnavailable as exc:
            self._log_fingerprint_failure(exc, exc.reason, trace, diarization, stream_index)
            raise
        except Exception as exc:  # noqa: BLE001 - library messages may carry paths or tokens
            reason = _classify(exc, "inference_failed")
            self._log_fingerprint_failure(exc, reason, trace, diarization, stream_index)
            raise VoiceProviderUnavailable(reason) from None
        log.emit(
            logging.INFO,
            "voice_fingerprint_inference_done",
            device=self._device,
            stream_index=stream_index,
            vectors=len(result["fingerprints"]),
            embedded=trace["embedded"],
            skipped_out_of_audio=trace["skipped_out_of_audio"],
            cut_at_audio_end=trace["cut_at_audio_end"],
            inference_ms=int((time.monotonic() - started) * 1000),
        )
        return result

    def _log_fingerprint_failure(self, exc: BaseException, reason: str, trace: dict[str, Any], diarization: dict[str, Any], stream_index: int) -> None:
        cause = _original(exc)
        lengths = [float(t["end"]) - float(t["start"]) for s in diarization.get("speakers", []) for t in s.get("turns", []) if "start" in t and "end" in t]
        samples, rate = trace.get("num_samples"), trace.get("sample_rate")
        log.emit(
            logging.WARNING,
            "voice_fingerprint_failed",
            error_code=reason,
            exc_class=type(cause).__name__,
            error_summary=error_summary(cause),
            stage=trace["stage"],
            device=self._device,
            stream_index=stream_index,
            speakers=len(diarization.get("speakers", [])),
            segments=len(lengths),
            embeddable_segments=sum(1 for length in lengths if length >= MIN_EMBEDDABLE_SECONDS),
            shortest_segment_s=round(min(lengths), 3) if lengths else None,
            longest_segment_s=round(max(lengths), 3) if lengths else None,
            embedded=trace["embedded"],
            skipped_out_of_audio=trace["skipped_out_of_audio"],
            cut_at_audio_end=trace["cut_at_audio_end"],
            segment_start=trace.get("segment_start"),
            segment_end=trace.get("segment_end"),
            sample_rate=rate,
            channels=trace.get("channels"),
            audio_duration_s=round(samples / rate, 3) if isinstance(samples, int) and isinstance(rate, int) and rate > 0 else None,
            input_shape=trace.get("input_shape"),
        )
