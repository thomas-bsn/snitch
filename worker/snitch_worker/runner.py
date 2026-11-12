"""Small HTTPS polling loop for the phase 2 fake provider."""

from __future__ import annotations

import logging
import json
import os
import re
import signal
import ssl
import subprocess
import sys
import tempfile
import threading
import time
import wave
import struct
from concurrent.futures import Future, ThreadPoolExecutor, wait
from typing import Any, Callable, Mapping, Protocol
from urllib.parse import urljoin, urlparse

import httpx
from snitch_client.api.default.claim_worker_job import sync as claim_job
from snitch_client.api.default.append_worker_job_logs import sync as append_job_logs
from snitch_client.api.default.complete_worker_job import sync as complete_job
from snitch_client.api.default.fail_worker_job import sync as fail_job
from snitch_client.api.default.get_worker_job_input import sync as get_job_input
from snitch_client.api.default.heartbeat_worker_job import sync as heartbeat_job
from snitch_client.api.default.post_worker_logs import sync_detailed as post_logs
from snitch_client.api.default.worker_pulse import sync as pulse
from snitch_client.client import AuthenticatedClient
from snitch_client.models.error import Error
from snitch_client.models.worker_failure_input import WorkerFailureInput
from snitch_client.models.worker_job_logs_input import WorkerJobLogsInput
from snitch_client.models.worker_log_batch import WorkerLogBatch
from snitch_client.models.worker_log_batch_lines_item import WorkerLogBatchLinesItem
from snitch_client.models.worker_pulse_input import WorkerPulseInput
from snitch_client.models.worker_transcription_reachability_input import WorkerTranscriptionReachabilityInput
from snitch_client.models.worker_result_input import WorkerResultInput
from snitch_client.models.worker_result_input_result import WorkerResultInputResult

from snitch_worker import acoustic_dedup, audio_diagnostic, benchmark_separation, cancellation, log, revocation
from snitch_worker import hf_token_sync, model_orders, voice_download, voice_models, voice_runtime
from snitch_worker import remote_control
from snitch_worker.waveform import attach_peaks
from snitch_worker.declared_models import declared_models
from snitch_worker.declared_machine import declared_machine
from snitch_worker import transcription_config
from snitch_worker.transcription_config import ReachabilityMonitor, active_configuration, sync_configuration
from snitch_worker.voice import DIARIZATION_STEP, VoiceProviderUnavailable, ensure_provider_ready, is_voice_step, process_voice_step, requires_audio


def fake_provider(step: str) -> dict[str, Any]:
    """Return fixture metadata only; phase 2 must not inspect task media."""
    if not step.startswith("fixture."):
        raise ValueError("fake provider only supports fixture steps")
    return {
        "provider": "fixture",
        "step": step,
    }


class TranscriptionError(RuntimeError):
    """A categorised Whisper failure. It carries a catalogue code and a status, never the provider text."""

    def __init__(self, code: str, http_status: int | None = None, url: str = "", excerpt: str = "") -> None:
        super().__init__(code)
        self.code = code
        self.http_status = http_status
        # What the Worker observed of its own call (ADR 0028); only log_whisper_failure writes them, scrubbed.
        self.method = "POST"
        self.url = url
        self.excerpt = excerpt


EXCERPT_LIMIT = 240


def log_whisper_failure(error: "TranscriptionError", latency_ms: int) -> None:
    """Log the Worker's view of a failed Whisper call: method, URL, status, latency and a bounded cleaned excerpt."""
    fields: dict[str, Any] = {"code": error.code, "method": error.method, "url": error.url, "latency_ms": latency_ms}
    if error.http_status is not None:
        fields["http_status"] = error.http_status
    if error.excerpt:
        fields["excerpt"] = log.scrub(error.excerpt)[:EXCERPT_LIMIT]
    log.emit(logging.WARNING, "whisper_request_failed", **fields)


# Provider reasons of a diarization step mapped to the catalogue; other reasons keep the generic code.
DIARIZATION_FAILURE_CODES = {
    "model_path_missing": "pyannote_model_missing",
    "model_access_denied": "pyannote_access_denied",
    "cuda_unavailable": "gpu_unavailable",
    "gpu_out_of_memory": "gpu_out_of_memory",
    "model_load_failed": "diarization_inference_failed",
    "inference_failed": "diarization_inference_failed",
}


def _is_diarization_step(step: str) -> bool:
    return bool(DIARIZATION_STEP.fullmatch(step))


def _cuda() -> Any:
    # Never imported here: only a model that already loaded PyTorch can have used the GPU.
    torch = sys.modules.get("torch")
    return torch.cuda if torch is not None and torch.cuda.is_available() else None


def _measured(step: str, compute: Callable[[], dict[str, Any]]) -> dict[str, Any]:
    """Run the work of a task once its audio is prepared. A benchmark task reports its duration (for a separation, the
    transcription and fingerprints of the sources included) and, under CUDA, the peak of GPU memory PyTorch allocated,
    in this process or in an engine subprocess (#403); without CUDA, or for a transcription done by the external
    service, the peak is absent."""
    if not step.startswith("benchmark."):
        return compute()
    if (device := _cuda()) is not None:
        device.reset_peak_memory_stats()
    voice_runtime.gpu_peak.bytes = None
    started = time.monotonic()
    result = compute()
    result["inference_seconds"] = round(time.monotonic() - started, 3)
    device = _cuda()
    peaks = [p for p in (voice_runtime.gpu_peak.bytes, device and int(device.max_memory_allocated())) if p is not None]
    if peaks and not step.startswith("benchmark.transcription."):
        result["gpu_peak_bytes"] = max(peaks)
    return result


def _voice_config(task_input: Any) -> dict[str, Any]:
    voice_value = getattr(task_input, "voice", None)
    voice_input = voice_value.to_dict() if hasattr(voice_value, "to_dict") else {}
    return voice_input.get("config") or {}


class Transcriber(Protocol):
    def transcribe(self, wav_path: str, vocabulary: list[dict[str, Any]] | None = None) -> dict[str, Any]: ...


class OpenAICompatibleTranscriber:
    """Configurable OpenAI-compatible provider; no model is selected by default."""

    def __init__(self, endpoint: str, model: str, api_key: str = "") -> None:
        if not endpoint.startswith(("http://", "https://")) or not model.strip():
            raise ValueError("transcription endpoint and model are required")
        self.endpoint = endpoint.rstrip("/")
        self.model = model.strip()
        self.api_key = api_key

    def transcribe(self, wav_path: str, vocabulary: list[dict[str, Any]] | None = None, language: str = "fr") -> dict[str, Any]:
        headers = {"Authorization": f"Bearer {self.api_key}"} if self.api_key else {}
        try:
            return self._transcribe(wav_path, vocabulary, language, headers)
        except TranscriptionError:
            raise
        except httpx.TimeoutException as exc:
            raise TranscriptionError("whisper_timeout") from exc
        except httpx.HTTPStatusError as exc:
            status = exc.response.status_code
            try:
                excerpt = " ".join(exc.response.text.split())[:1000]
            except (httpx.HTTPError, UnicodeError):
                excerpt = ""
            code = "whisper_model_missing" if status == 404 else "whisper_http_error"
            raise TranscriptionError(code, status, self.endpoint + "/v1/audio/transcriptions", excerpt) from exc
        except httpx.TransportError as exc:
            raise TranscriptionError("whisper_network") from exc
        except (TypeError, ValueError, KeyError, AttributeError) as exc:
            raise TranscriptionError("whisper_invalid_response") from exc

    def _transcribe(self, wav_path: str, vocabulary: list[dict[str, Any]] | None, language: str, headers: dict[str, str]) -> dict[str, Any]:
        # A refused lease stops waiting for the server and closes the connection (ADR 0029).
        http = httpx.Client(timeout=3600)
        with http, open(wav_path, "rb") as audio:
            response = cancellation.wait_for(lambda: http.post(
                self.endpoint + "/v1/audio/transcriptions",
                headers=headers,
                data={
                    "model": self.model,
                    "response_format": "verbose_json",
                    "timestamp_granularities[]": ["segment", "word"],
                    "language": language,
                    **({"prompt": "; ".join(
                        f"{term['term']} (variantes: {', '.join(term.get('variants', []))}; graphie préférée: {term['preferred_spelling']})"
                        for term in vocabulary
                    )} if vocabulary else {}),
                },
                files={"file": ("audio.wav", audio, "audio/wav")},
            ), on_cancel=http.close)
        response.raise_for_status()
        payload = response.json()
        if not isinstance(payload, dict) or not isinstance(
            payload.get("segments"), list
        ):
            raise TypeError("provider returned invalid timestamp data")
        segments = payload["segments"]
        words = payload.get("words", [])
        if not isinstance(words, list):
            raise TypeError("provider returned invalid word timestamps")
        if words and segments:
            for segment in segments:
                if not isinstance(segment, dict):
                    raise TypeError("provider returned invalid segment data")
                segment_words = [
                    word
                    for word in words
                    if isinstance(word, dict)
                    and word.get("start", -1) >= segment.get("start", 0) - 0.05
                    and word.get("end", 0) <= segment.get("end", 0) + 0.05
                ]
                if segment.get("text", "").strip() and not segment_words:
                    raise TypeError("provider omitted word-level timestamps")
                segment["words"] = segment_words
        else:
            for segment in segments:
                if not isinstance(segment, dict) or not isinstance(
                    segment.get("words"), list
                ):
                    raise TypeError("provider omitted word-level timestamps")
                if segment.get("text", "").strip() and not segment["words"]:
                    raise TypeError("provider omitted word-level timestamps")
        return {
            "provider": "openai-compatible",
            "model": self.model,
            "version": payload.get("model", self.model),
            "language": payload.get("language", ""),
            "segments": segments,
        }


def prepare_audio(media_path: str, wav_path: str, stream_index: int) -> None:
    if stream_index < 0:
        raise ValueError("invalid audio stream index")
    # ffmpeg is killed when the job's lease is refused.
    cancellation.run_process(
        [
            "ffmpeg",
            "-nostdin",
            "-v",
            "error",
            "-i",
            media_path,
            "-map",
            f"0:{stream_index}",
            "-vn",
            "-ac",
            "1",
            "-ar",
            "16000",
            "-c:a",
            "pcm_s16le",
            "-y",
            wav_path,
        ],
        timeout=3600,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def download_media(client: AuthenticatedClient, task_input: Any, lease_token: str, destination: str) -> None:
    media_url = task_input.media_url
    if media_url.startswith("/"):
        media_url = urljoin(client._base_url, media_url)
    with client.get_httpx_client().stream("GET", media_url, headers={"X-Worker-Lease-Token": lease_token}) as media:
        media.raise_for_status()
        with open(destination, "wb") as output:
            for chunk in media.iter_bytes():
                cancellation.check()
                output.write(chunk)


def configured_transcriber(model: str = "", override: str | None = None) -> OpenAICompatibleTranscriber:
    active = active_configuration()
    return OpenAICompatibleTranscriber(
        active.endpoint if active is not None else os.environ.get("TRANSCRIPTION_API_URL", ""),
        ((active.model if active is not None else os.environ.get("TRANSCRIPTION_MODEL", "").strip()) if override is None else override.strip()) or model.strip(),
        os.environ.get("TRANSCRIPTION_API_KEY", ""),
    )


def detect_voice_spans(wav_path: str, config: dict[str, Any]) -> list[tuple[float, float]]:
    """Energy VAD returning source-relative spans in seconds."""
    vad = config.get("vad") or {}
    if vad.get("mode", "off") == "off":
        with wave.open(wav_path, "rb") as audio:
            return [(0.0, audio.getnframes() / audio.getframerate())]
    if vad.get("mode") != "energy":
        raise ValueError("unsupported VAD mode")
    with wave.open(wav_path, "rb") as audio:
        rate = audio.getframerate()
        width = audio.getsampwidth()
        frame_ms = 20
        frame_size = max(1, rate * frame_ms // 1000)
        chunks: list[tuple[int, int, bool]] = []
        index = 0
        threshold = int(float(vad["threshold"]) * (2 ** (8 * width - 1)))
        while True:
            data = audio.readframes(frame_size)
            if not data:
                break
            frames = len(data) // width
            if width == 2:
                count = len(data) // 2
                values = struct.unpack("<" + "h" * count, data[:count * 2])
                rms = int((sum(sample * sample for sample in values) / max(1, count)) ** 0.5)
            else:
                rms = 0
            chunks.append((index, index + frames, rms >= threshold))
            index += frames
    min_speech = max(1, int(vad.get("min_speech_ms", 100)) // frame_ms)
    min_silence = max(1, int(vad.get("min_silence_ms", 300)) // frame_ms)
    padding = max(0, int(vad.get("padding_ms", 100))) / 1000
    active = [flag for _, _, flag in chunks]
    # Bridge short silence gaps before collecting spans.
    gap = 0
    for i, is_active in enumerate(active):
        if not is_active:
            gap += 1
        elif 0 < gap < min_silence:
            for j in range(i-gap, i):
                active[j] = True
            gap = 0
        else:
            gap = 0
    spans: list[tuple[float, float]] = []
    start: int | None = None
    for i, is_active in enumerate(active + [False]):
        if is_active and start is None:
            start = i
        if not is_active and start is not None:
            if i - start >= min_speech:
                begin = max(0.0, chunks[start][0] / rate - padding)
                end = min(index / rate, chunks[i-1][1] / rate + padding)
                spans.append((begin, end))
            start = None
    return spans


def audio_plan_spans(plan: dict[str, Any]) -> list[tuple[float, float]]:
    """Track-relative speech intervals pinned by the plan's diagnostic."""
    return [(float(item["start"]), float(item["end"])) for item in plan.get("intervals") or []]


def transcribe_prepared_audio(
    transcriber: Transcriber,
    wav_path: str,
    config: dict[str, Any],
    vocabulary: list[dict[str, Any]],
    spans: list[tuple[float, float]] | None = None,
    offset: float = 0.0,
) -> dict[str, Any]:
    """Transcribe the speech spans of a prepared track.

    ``spans`` (track-relative) replaces the energy VAD when a plan pins them.
    ``offset`` is the stream start in the clip: segments and words get the span
    start and the offset added exactly once, so results are on the clip timeline.
    """
    if spans is None:
        spans = detect_voice_spans(wav_path, config)
    results: list[dict[str, Any]] = []
    with tempfile.TemporaryDirectory(prefix="snitch-vad-") as temporary:
        for index, (start, end) in enumerate(spans):
            cancellation.check()
            chunk_path = os.path.join(temporary, f"speech-{index}.wav")
            with wave.open(wav_path, "rb") as source:
                rate, width, channels = source.getframerate(), source.getsampwidth(), source.getnchannels()
                source.setpos(int(start * rate))
                frames = source.readframes(max(1, int((end-start) * rate)))
                with wave.open(chunk_path, "wb") as chunk:
                    chunk.setnchannels(channels); chunk.setsampwidth(width); chunk.setframerate(rate); chunk.writeframes(frames)
            if isinstance(transcriber, OpenAICompatibleTranscriber):
                part = transcriber.transcribe(chunk_path, vocabulary, str(config.get("language", "fr")))
            else:
                part = transcriber.transcribe(chunk_path, vocabulary)
            for segment in part.get("segments", []):
                segment["start"] = float(segment["start"]) + start + offset
                segment["end"] = float(segment["end"]) + start + offset
                for word in segment.get("words", []):
                    word["start"] = float(word["start"]) + start + offset
                    word["end"] = float(word["end"]) + start + offset
            results.extend(part.get("segments", []))
    if not spans:
        return {"provider": "openai-compatible", "model": config.get("model") or getattr(transcriber, "model", "fixture"), "version": getattr(transcriber, "model", "fixture"), "language": config.get("language", "fr"), "segments": []}
    metadata = part if results else {"provider": "openai-compatible", "model": getattr(transcriber, "model", "fixture"), "version": getattr(transcriber, "model", "fixture"), "language": config.get("language", "fr")}
    if config.get("model"):
        metadata["model"] = config["model"]
    metadata["segments"] = results
    return metadata


class WorkerUnauthorized(RuntimeError):
    """Snitch refused the Worker token (401). Not a revocation until the narrow endpoint confirms it."""


def _require_response(value: Any, operation: str) -> Any:
    if isinstance(value, Error) and value.error == "worker_token_invalid":
        raise WorkerUnauthorized(f"worker API {operation} refused the token")
    if value is None or isinstance(value, Error):
        code = value.error if isinstance(value, Error) and isinstance(value.error, str) else ""
        log.emit(logging.ERROR, "worker_api_rejected", operation=operation, api_code=code or "empty_response")
        raise ApiError(operation, code)
    return value


# ADR 0029. Presence pulses from a dedicated thread, independently of jobs and leases.
PRESENCE_INTERVAL = 10.0
SUPERVISOR_HEALTH_PATH = "/tmp/snitch-worker-health.json"
# ADR 0029. The lease renewal is what hears a refusal from the server (job reassigned, cancelled,
# Worker revoked): renewing every 10 s cancels the job within 10 s. The server lease lasts 2
# minutes, which leaves about a dozen renewals for a network error to be retried.
HEARTBEAT_INTERVAL = 10.0
# Reconnection delays grow from 1 s and never exceed this cap.
PRESENCE_RETRY_CAP = 10.0
# Lease duration set by the server on claim and on each renewal (2 minutes).
LEASE_TTL = 120.0
# Heartbeat answers meaning the server no longer accepts this lease: reassigned, cancelled or revoked.
LEASE_REFUSALS = frozenset({"lease_not_owned", "worker_token_invalid"})


def presence_retry_delay(attempt: int) -> float:
    """Delay before retry number `attempt` (1-based): 1, 2, 4, 8 seconds, then the cap."""
    return min(PRESENCE_RETRY_CAP, 2.0 ** min(max(attempt - 1, 0), 4))


# Issue #321. While claiming keeps failing, one line at most per interval, plus the first failure and the recovery.
CLAIM_LOG_INTERVAL = 60.0
# Issue #321. After consecutive claiming failures the wait doubles from the poll interval and never exceeds this.
CLAIM_RETRY_CAP = 10.0


def _error_category(error: BaseException) -> str:
    """Stable category of a claiming failure; the exception text is never logged (it can carry URLs)."""
    if isinstance(error, WorkerUnauthorized):
        return "unauthorized"
    if isinstance(error, httpx.TimeoutException):
        return "timeout"
    if isinstance(error, httpx.HTTPStatusError):
        return "http_status"
    if isinstance(error, httpx.TransportError):
        return "network"
    if isinstance(error, (ValueError, KeyError, TypeError, AttributeError)):
        # A proxy page, or JSON the generated client cannot read (JSONDecodeError is a ValueError).
        return "invalid_response"
    if isinstance(error, ApiError):
        return "api_error"
    return "internal"


class ApiError(RuntimeError):
    """Snitch answered with an error, or an answer the generated client could not type. Carries its stable code."""

    def __init__(self, operation: str, code: str = "") -> None:
        super().__init__(f"worker API {operation} failed")
        self.code = code


class ClaimCycle:
    """The step of the current claiming cycle, so a failure names the logical endpoint that failed."""

    def __init__(self, health: "ClaimHealth | None" = None) -> None:
        self.operation = "pulse"
        self._health = health

    def answered(self) -> None:
        """Snitch answered the claim: claiming is operational again."""
        if self._health is not None:
            self._health.succeeded()


class ClaimHealth:
    """Health of one claiming loop, distinct from presence (a healthy pulse says nothing of it). Each loop has
    its own: a success on one loop must not hide, or reset the backoff of, failures on another.

    Failures are logged without secrets and without noise: the first one, then at most one line per
    CLAIM_LOG_INTERVAL while they continue, and the recovery with how long claiming was down."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._failures = 0
        self._failing_since = 0.0
        self._last_success = time.monotonic()
        self._last_logged = 0.0
        self._suppressed = 0

    def failed(self, operation: str, error: BaseException) -> int:
        """Record a failed cycle and return the number of consecutive failures."""
        now = time.monotonic()
        with self._lock:
            self._failures += 1
            failures = self._failures
            if failures == 1:
                self._failing_since = now
            due = failures == 1 or now - self._last_logged >= CLAIM_LOG_INTERVAL
            if not due:
                self._suppressed += 1
                return failures
            suppressed, self._suppressed, self._last_logged = self._suppressed, 0, now
            since_success = now - self._last_success
        fields: dict[str, Any] = {
            "operation": operation,
            "error_category": _error_category(error),
            "error_type": type(error).__name__,
            "attempt": failures,
            "since_last_success_s": round(since_success, 1),
        }
        if isinstance(error, httpx.HTTPStatusError):
            fields["http_status"] = error.response.status_code
        if isinstance(error, ApiError) and error.code:
            fields["error_code"] = error.code
        if suppressed:
            fields["suppressed"] = suppressed
        log.emit(logging.WARNING, "claim_poll_failed", **fields)
        return failures

    def succeeded(self) -> None:
        now = time.monotonic()
        with self._lock:
            failures, self._failures, self._suppressed = self._failures, 0, 0
            down_since = self._failing_since
            self._last_success = now
        if failures:
            log.emit(logging.INFO, "claim_poll_recovered", failures=failures, downtime_s=round(now - down_since, 1))


class LeaseWatch:
    """Set by the lease renewal thread when the job no longer owns its lease."""

    def __init__(self) -> None:
        self.lost = threading.Event()
        self.reason = ""

    def lose(self, reason: str) -> None:
        if not self.lost.is_set():
            self.reason = reason
            self.lost.set()


def _discard_lost_result(lease: LeaseWatch, context: dict[str, Any]) -> bool:
    """A job whose lease was lost records neither a result nor a failure. Always returns True (the job is done)."""
    log.emit(logging.WARNING, "job_result_discarded", reason=lease.reason, **context)
    return True


def supported_steps() -> list[str]:
    """The step families this Worker can read. A diagnostic is declared only where Torch can run the detector;
    Snitch still dispatches one only after this Worker reports the pinned model ready (ADR 0048)."""
    steps = ["fixture.*", "transcription.audio.*", "voice.*", "audio.multitrack.v1", "audio.dedup.*", "benchmark.transcription.*", "benchmark.diarization.*", "benchmark.separation.*"]
    if audio_diagnostic.available():
        steps.append("audio.diagnostic.*")
    return steps


# Pairs the acoustic comparison may examine in one job: beyond it the remaining pairs are kept whole
# (pair_budget_exceeded), never guessed. A pair costs about 20 to 110 ms (docs/acoustic-dedup.md).
DEDUP_MAX_PAIRS = 1000


def run_audio_dedup(client: AuthenticatedClient, task_input: Any, lease_token: str) -> dict[str, Any]:
    """Acoustic comparison of the retained tracks of a plan (ADR 0053/0054): decode each track once to 16 kHz
    mono PCM, then compare the overlapping segments of different tracks. The report is evidence only."""
    value = getattr(task_input, "audio_dedup", None)
    task = value.to_dict() if hasattr(value, "to_dict") else None
    if not task or not task.get("tracks"):
        raise ValueError("audio dedup task missing")
    with tempfile.TemporaryDirectory(prefix="snitch-dedup-") as temporary:
        original = os.path.join(temporary, "original.media")
        download_media(client, task_input, lease_token, original)
        events: list[dict[str, Any]] = []
        for track in task["tracks"]:
            cancellation.check()
            index = int(track["track_index"])
            wav_path = os.path.join(temporary, f"track-{index}.wav")
            prepare_audio(original, wav_path, index)
            for segment in track["segments"]:
                if not float(segment["end"]) > float(segment["start"]) >= 0:
                    continue
                events.append(
                    {
                        "id": f"{index}:{track['job_id']}:{segment['index']}",
                        "track_index": index,
                        "job_id": int(track["job_id"]),
                        "path": wav_path,
                        "clip_offset": float(track["start_seconds"]),
                        "start": float(segment["start"]),
                        "end": float(segment["end"]),
                    }
                )
        cancellation.check()
        return acoustic_dedup.run(
            {"schema_version": acoustic_dedup.SCHEMA_VERSION, "config": {"max_pairs": DEDUP_MAX_PAIRS}, "events": events},
            cancel=cancellation.check,
        )


def run_audio_diagnostic(client: AuthenticatedClient, task_input: Any, lease_token: str, step: str, detector: Any = None) -> dict[str, Any]:
    """Examine one real stream with the pinned offline detector and return an observation (ADR 0048).

    The result keeps the detector identity of this Worker: a plan made for another detector or configuration is a
    failure of the job, never a result recorded under a different identity."""
    value = getattr(task_input, "audio_diagnostic", None)
    plan = value.to_dict() if hasattr(value, "to_dict") else None
    index = int(step.rsplit(".", 1)[1])
    if not isinstance(plan, dict) or plan.get("track_index") != index:
        raise ValueError("audio diagnostic input does not match the step")
    if plan.get("detector_version") != audio_diagnostic.VERSION or plan.get("detector_config_version") != audio_diagnostic.CONFIG_VERSION:
        raise ValueError("audio diagnostic detector version is not supported by this Worker")
    with tempfile.TemporaryDirectory(prefix="snitch-diagnostic-") as temporary:
        original = os.path.join(temporary, "original.media")
        download_media(client, task_input, lease_token, original)
        cancellation.check()
        duration = plan.get("duration_seconds")
        offset = plan.get("start_seconds")
        return audio_diagnostic.diagnose(
            original, index, offset=offset if isinstance(offset, (int, float)) else None,
            expected_duration=duration if isinstance(duration, (int, float)) else None, detector=detector,
        )


def _send_pulse(client: AuthenticatedClient, capacity: int, reachability: ReachabilityMonitor | None = None) -> None:
    body = WorkerPulseInput(
        capacity=capacity,
        supported_steps=supported_steps(),
        models=declared_models(os.environ, references=voice_models.active_references()),
        machine=declared_machine(),
    )
    # The probe runs on its own thread: the pulse only reads its last result, never waits for it.
    snapshot = (reachability or transcription_config.reachability).snapshot()
    if snapshot is not None:
        # The generated client accepts an omitted field but not an explicit None.
        snapshot = {key: value for key, value in snapshot.items() if value is not None}
        body.transcription_reachability = WorkerTranscriptionReachabilityInput.from_dict(snapshot)
    _require_response(pulse(client=client, body=body), "pulse")


def _presence_loop(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    interval: float,
    guard: revocation.RevocationGuard | None = None,
    reachability: ReachabilityMonitor | None = None,
) -> None:
    failures = 0
    delay = interval
    last_success = 0.0
    while not stop.wait(delay):
        if guard is not None and guard.revoked.is_set():
            return
        # This timestamp proves the presence loop itself is alive even when Snitch
        # is temporarily unreachable. The host watchdog only repairs a stale loop.
        loop_at = time.time()
        try:
            temporary = SUPERVISOR_HEALTH_PATH + ".tmp"
            with open(temporary, "w", encoding="utf-8") as health_file:
                json.dump({"loop_at": loop_at, "last_success": last_success}, health_file)
                health_file.flush()
                os.fsync(health_file.fileno())
            os.replace(temporary, SUPERVISOR_HEALTH_PATH)
        except OSError:
            pass
        try:
            _send_pulse(client, capacity, reachability)
        except (httpx.HTTPError, RuntimeError):
            # Exception text is never logged: HTTP client errors can include URLs.
            failures += 1
            delay = presence_retry_delay(failures)
            log.emit(logging.WARNING, "presence_pulse_failed", attempt=failures)
            continue
        except Exception as error:  # noqa: BLE001 - a bug in a pulse must never kill presence for good
            failures += 1
            delay = presence_retry_delay(failures)
            log.emit(logging.ERROR, "presence_pulse_error", attempt=failures, error_type=type(error).__name__)
            continue
        if failures:
            log.emit(logging.INFO, "presence_reconnected", attempts=failures)
            failures = 0
        delay = interval
        last_success = time.time()
        # Health reporting is diagnostic only; it must never affect job processing.


def sync_models_once(client: AuthenticatedClient) -> None:
    """Pull what Snitch holds for this Worker and run its orders: each step fails on its own, none stops the other."""
    try:
        voice_models.sync_voice_models(client, voice_models.default_manager())
    except Exception as error:  # noqa: BLE001 - the model manager must never stop the orders
        log.emit(logging.WARNING, "voice_model_sync_failed", error_type=type(error).__name__)
    try:
        model_orders.sync_orders(client, model_orders.default_executor())
    except Exception as error:  # noqa: BLE001 - one refused pull must not stop the next
        log.emit(logging.WARNING, "voice_order_sync_failed", error_type=type(error).__name__)


def _models_loop(client: AuthenticatedClient, stop: threading.Event, interval: float, guard: revocation.RevocationGuard | None = None) -> None:
    """Orders are pulled on their own thread: a job that runs for hours never leaves an order pending (#327).

    The loop outlives every recoverable error, like the claiming loops (#321)."""
    while not stop.is_set():
        if guard is not None and guard.revoked.is_set():
            return
        if isinstance(client, AuthenticatedClient):
            sync_models_once(client)
        stop.wait(interval)


def _heartbeat_loop(
    client: AuthenticatedClient,
    job_id: int,
    lease_token: str,
    interval: float,
    stop: threading.Event,
    lease: LeaseWatch,
    context: dict[str, Any] | None = None,
) -> None:
    renewed_at = time.monotonic()
    delay = interval
    attempt = 0
    while not stop.wait(delay):
        try:
            value = heartbeat_job(
                client=client,
                id=job_id,
                x_worker_lease_token=lease_token,
            )
        except (httpx.HTTPError, KeyError, TypeError, ValueError, AttributeError):
            # A proxy may answer HTML or a different JSON shape while Snitch restarts.
            # Treat an unreadable API response like a network outage, not a lost thread.
            value = None
        if isinstance(value, Error) and value.error in LEASE_REFUSALS:
            log.emit(logging.WARNING, "job_lease_refused", error_code=value.error, **(context or {}))
            lease.lose("refused")
            return
        if value is None or isinstance(value, Error):
            # Transient failure: retry until the lease would have expired.
            attempt += 1
            remaining = renewed_at + LEASE_TTL - time.monotonic()
            if remaining <= 0:
                log.emit(logging.WARNING, "job_lease_expired", **(context or {}))
                lease.lose("expired")
                return
            delay = min(presence_retry_delay(attempt), remaining)
            continue
        attempt = 0
        renewed_at = time.monotonic()
        delay = interval
        log.emit(logging.DEBUG, "job_heartbeat", **(context or {}))


def run_once(
    client: AuthenticatedClient,
    capacity: int = 1,
    heartbeat_interval: float = HEARTBEAT_INTERVAL,
    stop: threading.Event | None = None,
    transcriber: Transcriber | None = None,
    voice_providers: Mapping[str, Any] | None = None,
    cycle: ClaimCycle | None = None,
) -> bool:
    """Process at most one leased task. Return False when the queue is empty.

    ``cycle`` follows the step in progress, so the claiming loop can name the one that failed."""
    if heartbeat_interval <= 0:
        raise ValueError("heartbeat interval must be positive")
    cycle = cycle or ClaimCycle()
    cycle.operation = "pulse"
    try:
        _send_pulse(client, capacity)
    except WorkerUnauthorized:
        raise
    except Exception as error:  # noqa: BLE001 - the presence thread keeps the canonical pulse (#321)
        # Claiming never depends on this pulse: a pulse Snitch keeps refusing must not stop the claims
        # while the other calls keep the Worker Online. Presence logs its own failures.
        log.emit(logging.DEBUG, "claim_cycle_pulse_failed", error_category=_error_category(error), error_type=type(error).__name__)
    if stop is not None and stop.is_set():
        return False
    if isinstance(client, AuthenticatedClient):
        # Kept blocking: a transcription job is only taken with the server's configuration validated here.
        cycle.operation = "transcription_config"
        sync_configuration(client)
        try:
            hf_token_sync.sync(client, voice_download.default_token_file(os.environ))
        except Exception:  # noqa: BLE001 - the token sync must never stop claiming, and its error text may carry the value
            log.emit(logging.WARNING, "hf_token_sync_unavailable")
    cycle.operation = "claim"
    claim = _require_response(claim_job(client=client), "claim")
    if claim.job is not None and not hasattr(claim.job, "id"):
        # The generated client keeps a job it could not type as a plain value: never run an unreadable job.
        raise ApiError("claim", "unreadable_job")
    # Claiming works again from here, whatever the job then does.
    cycle.answered()
    cycle.operation = "job"
    if claim.job is None:
        log.emit(logging.DEBUG, "claim_empty")
        return False
    # The Worker's log lines of this job are shipped in batches while the job runs (ADR 0028).
    job_id = claim.job.id
    shipper = log.LineShipper(job_id, lambda lines: _send_job_logs(client, job_id, claim.lease_token, lines))
    log.get_logger().addHandler(shipper)
    try:
        return _run_claimed(client, claim, heartbeat_interval, transcriber, voice_providers, shipper)
    finally:
        log.get_logger().removeHandler(shipper)
        shipper.close()


def _send_job_logs(client: AuthenticatedClient, job_id: int, lease_token: str, lines: list[str]) -> None:
    _require_response(
        append_job_logs(client=client, id=job_id, body=WorkerJobLogsInput(lines=lines), x_worker_lease_token=lease_token),
        "logs",
    )


def _run_claimed(
    client: AuthenticatedClient,
    claim: Any,
    heartbeat_interval: float,
    transcriber: Transcriber | None,
    voice_providers: Mapping[str, Any] | None,
    shipper: log.LineShipper,
) -> bool:
    """Run the claimed job to its end and report it. Always True: a job was taken.

    Log lines are flushed before the outcome is reported: once the job is completed or failed the
    lease is gone and the server no longer accepts lines for it."""
    job = claim.job
    lease_token = claim.lease_token
    # Correlation context taken from the claim; the lease token is never part of it.
    context = {
        "clip_id": getattr(job, "clip_id", None),
        "job_id": job.id,
        "step": job.step,
        "cycle": getattr(job, "cycle", None),
        "attempt": getattr(job, "attempts", None),
        "worker_id": getattr(job, "lease_owner", None),
    }
    started = time.monotonic()
    log.emit(logging.INFO, "job_claimed", **context)
    active_transcriber: Transcriber | None = None
    heartbeat_stop = threading.Event()
    lease = LeaseWatch()
    heartbeater = threading.Thread(
        target=_heartbeat_loop,
        args=(
            client,
            job.id,
            lease_token,
            heartbeat_interval,
            heartbeat_stop,
            lease,
            context,
        ),
        daemon=True,
    )
    heartbeater.start()
    log.emit(logging.INFO, "job_started", **context)
    try:
        # A refused lease cancels the work in progress (ADR 0029, cancellation.py).
        with cancellation.scope(lease.lost):
            task_input = _require_response(
                get_job_input(
                    client=client,
                    id=job.id,
                    x_worker_lease_token=lease_token,
                ),
                "task input",
            )
            if job.step.startswith("fixture."):
                result = fake_provider(job.step)
            elif re.fullmatch(r"audio\.dedup\.\d+", job.step):
                result = run_audio_dedup(client, task_input, lease_token)
            elif re.fullmatch(r"audio\.diagnostic\.\d+", job.step):
                result = run_audio_diagnostic(client, task_input, lease_token, job.step)
            elif re.fullmatch(r"transcription\.audio\.\d+|benchmark\.transcription\.\d+", job.step):
                # A benchmark task transcribes like a production one, with the model the task asks for (ADR 0058).
                with tempfile.TemporaryDirectory(
                    prefix="snitch-transcription-"
                ) as temporary:
                    original = os.path.join(temporary, "original.media")
                    wav_path = os.path.join(temporary, "prepared.wav")
                    download_media(client, task_input, lease_token, original)
                    stream_index = int(job.step.rsplit(".", 1)[1])
                    cancellation.check()
                    prepare_audio(original, wav_path, stream_index)
                    cancellation.check()
                    pipeline_value = getattr(task_input, "pipeline", None)
                    pipeline = pipeline_value.to_dict() if hasattr(pipeline_value, "to_dict") else (pipeline_value or {})
                    override = getattr(task_input, "transcription_model", None)
                    if not isinstance(override, str):
                        override = None
                    active_transcriber = transcriber or configured_transcriber(str(pipeline.get("model", "")), override)
                    vocabulary = [
                        {"term": term.term, "variants": term.variants, "preferred_spelling": term.preferred_spelling}
                        for term in (task_input.vocabulary or [])
                    ]
                    cancellation.check()
                    plan_value = getattr(task_input, "audio_plan", None)
                    plan = plan_value.to_dict() if hasattr(plan_value, "to_dict") else None
                    if plan:
                        if int(plan["track_index"]) != stream_index:
                            raise ValueError("audio plan track does not match the job")
                        # A plan without usable intervals (forced track) falls back to the VAD.
                        pinned = audio_plan_spans(plan) or None
                        result = transcribe_prepared_audio(
                            active_transcriber, wav_path, pipeline, vocabulary, pinned, float(plan["start_seconds"])
                        )
                        result.update(
                            audio_stream_index=stream_index,
                            time_base="clip",
                            audio_plan_id=plan["plan_id"],
                            audio_plan_version=plan["input_version"],
                        )
                    else:
                        result = _measured(job.step, lambda: transcribe_prepared_audio(active_transcriber, wav_path, pipeline, vocabulary))
                        result["audio_stream_index"] = stream_index
                        attach_peaks(result, wav_path)
            elif re.fullmatch(r"benchmark\.separation\.\d+", job.step):
                # The separation of a benchmark chain (#402): its result is only stored by Snitch, never projected.
                with tempfile.TemporaryDirectory(prefix="snitch-separation-") as temporary:
                    original = os.path.join(temporary, "original.media")
                    wav_path = os.path.join(temporary, "prepared.wav")
                    download_media(client, task_input, lease_token, original)
                    stream_index = int(job.step.rsplit(".", 1)[1])
                    prepare_audio(original, wav_path, stream_index)
                    cancellation.check()
                    pipeline_value = getattr(task_input, "pipeline", None)
                    pipeline = pipeline_value.to_dict() if hasattr(pipeline_value, "to_dict") else (pipeline_value or {})
                    active_transcriber = transcriber or configured_transcriber(str(pipeline.get("model", "")), task_input.transcription_model)
                    result = _measured(job.step, lambda: benchmark_separation.run(
                        task_input.voice.to_dict(), task_input.separation.to_dict(), wav_path, stream_index,
                        active_transcriber, pipeline, voice_providers, os.environ))
            elif is_voice_step(job.step) or re.fullmatch(r"benchmark\.diarization\.\d+", job.step):
                # A benchmark diarization runs the production code on the model the task asks for, under the name of
                # the voice step it imitates; it computes no waveform peaks (ADR 0059).
                benchmark = job.step.startswith("benchmark.")
                voice_step = "voice." + job.step[len("benchmark."):] if benchmark else job.step
                voice_value = getattr(task_input, "voice", None)
                voice_input = voice_value.to_dict() if hasattr(voice_value, "to_dict") else {}
                duration = getattr(task_input, "duration_seconds", None)
                duration = duration if isinstance(duration, (int, float)) else None
                voice_config = voice_input.get("config") or {}
                if requires_audio(voice_config):
                    # Fail on a missing model path before downloading any media.
                    ensure_provider_ready(voice_input, voice_providers, step=voice_step)
                    with tempfile.TemporaryDirectory(prefix="snitch-voice-") as temporary:
                        original = os.path.join(temporary, "original.media")
                        wav_path = os.path.join(temporary, "prepared.wav")
                        download_media(client, task_input, lease_token, original)
                        prepare_audio(original, wav_path, int(job.step.rsplit(".", 1)[1]))
                        result = _measured(job.step, lambda: process_voice_step(voice_step, voice_input, duration, wav_path, voice_providers))
                        if _is_diarization_step(voice_step) and not benchmark:
                            attach_peaks(result, wav_path)
                else:
                    # No audio is prepared here, so no peaks: a result without them is accepted (ADR 0034).
                    result = process_voice_step(voice_step, voice_input, duration, None, voice_providers)
            else:
                raise ValueError("unsupported worker step")
        heartbeat_stop.set()
        heartbeater.join()
        if lease.lost.is_set():
            # ADR 0029: a job whose lease was refused or expired sends no result.
            return _discard_lost_result(lease, context)
        if _is_diarization_step(job.step):
            config = _voice_config(task_input)
            speakers = result.get("speakers") or []
            log.emit(
                logging.INFO,
                "diarization_done",
                provider=config.get("provider"),
                model=config.get("model"),
                revision=config.get("model_version"),
                speakers=len(speakers),
                segments=sum(len(speaker.get("turns") or []) for speaker in speakers),
                duration_ms=int((time.monotonic() - started) * 1000),
                **context,
            )
        shipper.flush()
        result_model = WorkerResultInputResult()
        result_model.additional_properties.update(result)
        _require_response(
            complete_job(
                client=client,
                id=job.id,
                body=WorkerResultInput(
                    input_version=job.input_version, result=result_model
                ),
                x_worker_lease_token=lease_token,
            ),
            "complete",
        )
        log.emit(logging.INFO, "job_completed", duration_ms=int((time.monotonic() - started) * 1000), **context)
    except cancellation.JobCancelled:
        # The lease was refused while the job ran: the work stopped, nothing is recorded.
        heartbeat_stop.set()
        heartbeater.join()
        log.emit(logging.WARNING, "job_cancelled", reason=lease.reason, duration_ms=int((time.monotonic() - started) * 1000), **context)
    except Exception as exc:  # noqa: BLE001 - any error of the job is a recorded failure, never a lost claiming thread (#321)
        heartbeat_stop.set()
        heartbeater.join()
        if lease.lost.is_set():
            # The job was reassigned or expired: neither a result nor a failure may be recorded.
            return _discard_lost_result(lease, context)
        diagnostic: dict[str, Any] = {}
        if isinstance(exc, TranscriptionError):
            error_code = exc.code
            diagnostic = {"provider": "openai-compatible", "duration_ms": int((time.monotonic() - started) * 1000)}
            model = getattr(transcriber, "model", None) or getattr(active_transcriber, "model", None)
            if isinstance(model, str) and model:
                diagnostic["model"] = model
            if exc.http_status is not None:
                diagnostic["http_status"] = exc.http_status
            log_whisper_failure(exc, diagnostic["duration_ms"])
        elif isinstance(exc, VoiceProviderUnavailable) and _is_diarization_step(job.step) and exc.reason in DIARIZATION_FAILURE_CODES:
            error_code = DIARIZATION_FAILURE_CODES[exc.reason]
            voice_config = _voice_config(task_input)
            diagnostic = {"duration_ms": int((time.monotonic() - started) * 1000)}
            for field, key in (("provider", "provider"), ("model", "model"), ("revision", "model_version")):
                value = voice_config.get(key)
                if isinstance(value, str) and value:
                    diagnostic[field] = value
        elif isinstance(exc, VoiceProviderUnavailable) and exc.reason == "gpu_out_of_memory" and job.step.startswith("benchmark.separation."):
            error_code = "gpu_out_of_memory"  # Snitch does not retry it: the same window would fail again
        else:
            error_code = (
                "audio_preparation_failed"
                if isinstance(exc, (OSError, subprocess.SubprocessError))
                else "provider_unavailable"
            )
        # Only the code and the exception class: the raw text can echo URLs, paths or tokens.
        log.emit(
            logging.WARNING,
            "job_failed",
            error_code=error_code,
            error_class=type(exc).__name__,
            duration_ms=int((time.monotonic() - started) * 1000),
            **context,
        )
        shipper.flush()
        try:
            _require_response(
                fail_job(
                    client=client,
                    id=job.id,
                    body=WorkerFailureInput(error=error_code, **diagnostic),
                    x_worker_lease_token=lease_token,
                ),
                "fail",
            )
        except Exception:  # noqa: BLE001 - an unrecorded failure only lets the lease expire
            # The lease may have expired or moved while the provider was busy.
            # In that case the old attempt must not mutate the recovered task.
            log.emit(logging.WARNING, "job_failure_not_recorded", **context)
    finally:
        # Whatever ends the job, its lease is no longer renewed: a renewal left running would keep
        # the job leased forever by a Worker that no longer works on it.
        heartbeat_stop.set()
    return True


def _worker_loop(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float,
    heartbeat_interval: float,
    guard: revocation.RevocationGuard | None = None,
    health: ClaimHealth | None = None,
    control: remote_control.RemoteControl | None = None,
) -> None:
    """Claim until stopped. No recoverable error ends this loop (#321): a redeploy, a proxy page, a stale
    socket or an answer the generated client cannot read is logged and retried with a bounded backoff."""
    health = health or ClaimHealth()
    failures = 0
    while not stop.is_set():
        if guard is not None and guard.revoked.is_set():
            return
        if control is not None and control.paused.is_set():
            # Paused by an administrator (ADR 0063): nothing new is claimed, a job already running is not touched.
            stop.wait(poll_interval)
            continue
        cycle = ClaimCycle(health)
        try:
            completed = run_once(client, capacity, heartbeat_interval, stop, cycle=cycle)
            failures = 0
        except WorkerUnauthorized as error:
            if guard is not None and guard.on_unauthorized(client):
                stop.set()
                return
            failures = health.failed(cycle.operation, error)
            completed = False
        except (Exception, cancellation.JobCancelled) as error:  # noqa: BLE001 - see the docstring
            # Exception text is never logged: HTTP client errors can include URLs.
            failures = health.failed(cycle.operation, error)
            completed = False
        if not completed:
            stop.wait(claim_retry_delay(failures, poll_interval))


def claim_retry_delay(failures: int, poll_interval: float) -> float:
    """The poll interval while claiming works; after consecutive failures, doubled up to a bounded wait."""
    if failures <= 0:
        return poll_interval
    return min(max(poll_interval, CLAIM_RETRY_CAP), poll_interval * 2.0 ** min(failures - 1, 10))


def run_until_stopped(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float = 5.0,
    heartbeat_interval: float = HEARTBEAT_INTERVAL,
    models_dir: str | None = None,
    presence_interval: float = PRESENCE_INTERVAL,
    reachability: ReachabilityMonitor | None = None,
    control: remote_control.RemoteControl | None = None,
) -> bool:
    """Run a bounded pool; stop claiming on shutdown and drain active tasks.

    Presence runs on its own thread, so a long job never makes the Worker look Offline.
    Return True when Snitch confirmed this Worker's deletion (the models volume is then erased)."""
    if capacity < 1 or capacity > 4:
        raise ValueError("worker capacity must be an integer from 1 to 4")
    if poll_interval <= 0 or heartbeat_interval <= 0 or presence_interval <= 0:
        raise ValueError("worker intervals must be positive")

    guard = revocation.RevocationGuard(models_dir or revocation.models_dir(os.environ))
    monitor = reachability or transcription_config.reachability
    presence = threading.Thread(
        target=_presence_loop,
        args=(client, capacity, stop, presence_interval, guard, monitor),
        daemon=True,
        name="snitch-presence",
    )
    probe = threading.Thread(target=monitor.run, args=(stop,), daemon=True, name="snitch-reachability")
    orders = threading.Thread(target=_models_loop, args=(client, stop, poll_interval, guard), daemon=True, name="snitch-model-orders")
    controller = None
    if control is not None:
        # The wanted state is read once before the first claim, so a paused Worker that restarts stays paused.
        try:
            control.sync_once(client)
        except Exception as error:  # noqa: BLE001 - Snitch out of reach at start: the loop below keeps trying
            log.emit(logging.WARNING, "worker_control_unreachable", error_type=type(error).__name__)
        controller = threading.Thread(target=remote_control.control_loop, args=(client, control, stop, poll_interval), daemon=True, name="snitch-remote-control")
    presence.start()
    probe.start()
    orders.start()
    if controller is not None:
        controller.start()
    with ThreadPoolExecutor(max_workers=capacity) as executor:

        def launch() -> Future[None]:
            return executor.submit(_worker_loop, client, capacity, stop, poll_interval, heartbeat_interval, guard, ClaimHealth(), control)

        workers = [launch() for _ in range(capacity)]
        restarts = [0] * capacity
        restart_at: dict[int, float] = {}
        while not stop.wait(0.1):
            # Supervision (#321): a claiming loop that ended while the Worker runs is reported and restarted
            # in-process, with the same client and identity, without relying on an external watchdog.
            for index, worker in enumerate(workers):
                if index in restart_at:
                    if time.monotonic() >= restart_at[index] and not stop.is_set() and not guard.revoked.is_set():
                        del restart_at[index]
                        workers[index] = launch()
                    continue
                if not worker.done() or guard.revoked.is_set():
                    continue
                error = worker.exception()
                restarts[index] += 1
                log.emit(
                    logging.ERROR,
                    "claim_loop_stopped",
                    error_type=type(error).__name__ if error is not None else "none",
                    restarts=restarts[index],
                )
                restart_at[index] = time.monotonic() + claim_retry_delay(restarts[index], poll_interval)
        wait(workers)
    presence.join()
    probe.join()
    orders.join()
    if controller is not None:
        controller.join()
    return guard.revoked.is_set()


def _push_logs(client: AuthenticatedClient, session_id: str, lines: list[tuple[int, str]]) -> None:
    """Hand a batch of the Worker's own journal to Snitch. Raises when Snitch did not take it."""
    body = WorkerLogBatch(session_id=session_id, lines=[WorkerLogBatchLinesItem(seq=seq, line=line) for seq, line in lines])
    answer = post_logs(client=client, body=body)
    status = int(answer.status_code)
    if 200 <= status < 300:
        return
    # A 4xx other than an unknown or expired token (401), a throttle (429) or a timeout (408) is a refusal of
    # the batch itself: it would be refused again, so it is dropped rather than resent for ever.
    if 400 <= status < 500 and status not in (401, 403, 404, 408, 429):
        raise log.BatchRefused()
    raise RuntimeError("worker logs not taken")


def main() -> None:
    log.configure()
    # The journal is sent from the very first line, so an administrator reads it from the start of the process.
    uplink = log.WorkerLogUplink()
    log.get_logger().addHandler(uplink)
    shutdown = threading.Event()

    def request_shutdown(_signum: int, _frame: object) -> None:
        shutdown.set()

    models_dir = revocation.models_dir(os.environ)
    if revocation.is_revoked(models_dir):
        # Deleted in Snitch earlier: no client, no network call. Staying alive keeps the restart
        # policy from looping, and a valid Worker (no marker) still restarts after a host reboot.
        signal.signal(signal.SIGTERM, request_shutdown)
        signal.signal(signal.SIGINT, request_shutdown)
        revocation.wait_inert(shutdown)
        return
    api_url = os.environ.get("SNITCH_API_URL", os.environ.get("RECEIPTS_API_URL", "")).strip()
    token = os.environ.get("SNITCH_WORKER_TOKEN", os.environ.get("RECEIPTS_WORKER_TOKEN", "")).strip()
    capacity_text = os.environ.get("SNITCH_WORKER_CAPACITY", os.environ.get("RECEIPTS_WORKER_CAPACITY", "1"))
    if (
        not api_url
        or urlparse(api_url).scheme != "https"
        or not urlparse(api_url).netloc
    ):
        raise SystemExit("SNITCH_API_URL must be an HTTPS URL")
    if not token:
        raise SystemExit("SNITCH_WORKER_TOKEN is required")
    try:
        capacity = int(capacity_text)
    except ValueError as exc:
        raise SystemExit(
            "SNITCH_WORKER_CAPACITY must be an integer from 1 to 4"
        ) from exc
    if capacity < 1 or capacity > 4:
        raise SystemExit("SNITCH_WORKER_CAPACITY must be an integer from 1 to 4")
    ca_bundle = os.environ.get("SNITCH_WORKER_CA_BUNDLE", os.environ.get("RECEIPTS_WORKER_CA_BUNDLE"))
    tls_context = ssl.create_default_context(cafile=ca_bundle) if ca_bundle else True
    with AuthenticatedClient(
        base_url=api_url.rstrip("/"),
        token=token,
        verify_ssl=tls_context,
        timeout=httpx.Timeout(30.0),
    ) as client:
        stop = threading.Event()

        def request_stop(_signum: int, _frame: object) -> None:
            stop.set()
            shutdown.set()

        signal.signal(signal.SIGTERM, request_stop)
        signal.signal(signal.SIGINT, request_stop)
        uplink.start(lambda session_id, lines: _push_logs(client, session_id, lines))
        revoked = run_until_stopped(client, capacity, stop, models_dir=models_dir, control=remote_control.RemoteControl())
        uplink.close()
    if revoked:
        revocation.wait_inert(shutdown)


if __name__ == "__main__":
    main()
