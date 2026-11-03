"""Small HTTPS polling loop for the phase 2 fake provider."""

from __future__ import annotations

import logging
import os
import re
import signal
import ssl
import subprocess
import tempfile
import threading
import time
import wave
import struct
from concurrent.futures import ThreadPoolExecutor
from typing import Any, Mapping, Protocol
from urllib.parse import urljoin, urlparse

import httpx
from snitch_client.api.default.claim_worker_job import sync as claim_job
from snitch_client.api.default.append_worker_job_logs import sync as append_job_logs
from snitch_client.api.default.complete_worker_job import sync as complete_job
from snitch_client.api.default.fail_worker_job import sync as fail_job
from snitch_client.api.default.get_worker_job_input import sync as get_job_input
from snitch_client.api.default.heartbeat_worker_job import sync as heartbeat_job
from snitch_client.api.default.worker_pulse import sync as pulse
from snitch_client.client import AuthenticatedClient
from snitch_client.models.error import Error
from snitch_client.models.worker_failure_input import WorkerFailureInput
from snitch_client.models.worker_job_logs_input import WorkerJobLogsInput
from snitch_client.models.worker_pulse_input import WorkerPulseInput
from snitch_client.models.worker_result_input import WorkerResultInput
from snitch_client.models.worker_result_input_result import WorkerResultInputResult

from snitch_worker import log, revocation
from snitch_worker import voice_models
from snitch_worker.declared_models import declared_models
from snitch_worker.declared_machine import declared_machine
from snitch_worker.transcription_config import active_configuration, sync_configuration
from snitch_worker.voice import DIARIZATION_STEP, TRIAL_STEP, VoiceProviderUnavailable, ensure_provider_ready, is_trial_step, is_voice_step, process_trial_step, process_voice_step, requires_audio


def trial_voice_input(task_input: Any) -> dict[str, Any]:
    """A diarization trial's own input (``voice_trial``): its pinned model, options and audio stream.

    It is distinct from the regular pipeline input (``voice``), which a trial never receives.
    """
    value = getattr(task_input, "voice_trial", None)
    return value.to_dict() if hasattr(value, "to_dict") else {}


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

    def __init__(self, code: str, http_status: int | None = None) -> None:
        super().__init__(code)
        self.code = code
        self.http_status = http_status


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
    return bool(DIARIZATION_STEP.fullmatch(step) or TRIAL_STEP.fullmatch(step))


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
            raise TranscriptionError("whisper_model_missing" if status == 404 else "whisper_http_error", status) from exc
        except httpx.TransportError as exc:
            raise TranscriptionError("whisper_network") from exc
        except (TypeError, ValueError, KeyError, AttributeError) as exc:
            raise TranscriptionError("whisper_invalid_response") from exc

    def _transcribe(self, wav_path: str, vocabulary: list[dict[str, Any]] | None, language: str, headers: dict[str, str]) -> dict[str, Any]:
        with open(wav_path, "rb") as audio:
            response = httpx.post(
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
                timeout=3600,
            )
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
    subprocess.run(
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
        check=True,
        timeout=3600,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
    )


def download_media(client: AuthenticatedClient, task_input: Any, lease_token: str, destination: str) -> None:
    media_url = task_input.media_url
    if media_url.startswith("/"):
        media_url = urljoin(client._base_url, media_url)
    with client.get_httpx_client().stream("GET", media_url, headers={"X-Worker-Lease-Token": lease_token}) as media:
        media.raise_for_status()
        with open(destination, "wb") as output:
            output.writelines(media.iter_bytes())


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


def transcribe_prepared_audio(transcriber: Transcriber, wav_path: str, config: dict[str, Any], vocabulary: list[dict[str, Any]]) -> dict[str, Any]:
    spans = detect_voice_spans(wav_path, config)
    results: list[dict[str, Any]] = []
    with tempfile.TemporaryDirectory(prefix="snitch-vad-") as temporary:
        for index, (start, end) in enumerate(spans):
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
                segment["start"] = float(segment["start"]) + start
                segment["end"] = float(segment["end"]) + start
                for word in segment.get("words", []):
                    word["start"] = float(word["start"]) + start
                    word["end"] = float(word["end"]) + start
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
        raise RuntimeError(f"worker API {operation} failed")
    return value


# ADR 0029. Presence pulses from a dedicated thread, independently of jobs and leases.
PRESENCE_INTERVAL = 10.0
# Reconnection delays grow from 1 s and never exceed this cap.
PRESENCE_RETRY_CAP = 10.0
# Lease duration set by the server on claim and on each renewal (2 minutes).
LEASE_TTL = 120.0
# Heartbeat answers meaning the server no longer accepts this lease: reassigned, cancelled or revoked.
LEASE_REFUSALS = frozenset({"lease_not_owned", "worker_token_invalid"})


def presence_retry_delay(attempt: int) -> float:
    """Delay before retry number `attempt` (1-based): 1, 2, 4, 8 seconds, then the cap."""
    return min(PRESENCE_RETRY_CAP, 2.0 ** (attempt - 1))


class LeaseWatch:
    """Set by the lease renewal thread when the job no longer owns its lease."""

    def __init__(self) -> None:
        self.lost = threading.Event()
        self.reason = ""

    def lose(self, reason: str) -> None:
        if not self.lost.is_set():
            self.reason = reason
            self.lost.set()


def _send_pulse(client: AuthenticatedClient, capacity: int) -> None:
    _require_response(
        pulse(
            client=client,
            body=WorkerPulseInput(
                capacity=capacity,
                supported_steps=["fixture.*", "transcription.audio.*", "voice.*", "voicetrial.*"],
                models=declared_models(os.environ, voice_models.active_repo()),
                machine=declared_machine(),
            ),
        ),
        "pulse",
    )


def _presence_loop(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    interval: float,
    guard: revocation.RevocationGuard | None = None,
) -> None:
    failures = 0
    delay = interval
    while not stop.wait(delay):
        if guard is not None and guard.revoked.is_set():
            return
        try:
            _send_pulse(client, capacity)
        except (httpx.HTTPError, RuntimeError):
            # Exception text is never logged: HTTP client errors can include URLs.
            failures += 1
            delay = presence_retry_delay(failures)
            log.emit(logging.WARNING, "presence_pulse_failed", attempt=failures)
            continue
        if failures:
            log.emit(logging.INFO, "presence_reconnected", attempts=failures)
            failures = 0
        delay = interval


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
        except httpx.HTTPError:
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
    heartbeat_interval: float = 45.0,
    stop: threading.Event | None = None,
    transcriber: Transcriber | None = None,
    voice_providers: Mapping[str, Any] | None = None,
) -> bool:
    """Process at most one leased task. Return False when the queue is empty."""
    if heartbeat_interval <= 0:
        raise ValueError("heartbeat interval must be positive")
    _send_pulse(client, capacity)
    if stop is not None and stop.is_set():
        return False
    if isinstance(client, AuthenticatedClient):
        sync_configuration(client)
        try:
            voice_models.sync_voice_models(client, voice_models.default_manager())
        except Exception:  # noqa: BLE001 - the model manager must never stop claiming
            log.emit(logging.WARNING, "voice_model_sync_failed")
    claim = _require_response(claim_job(client=client), "claim")
    if claim is None or claim.job is None:
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
        elif re.fullmatch(r"transcription\.audio\.\d+", job.step):
            with tempfile.TemporaryDirectory(
                prefix="snitch-transcription-"
            ) as temporary:
                original = os.path.join(temporary, "original.media")
                wav_path = os.path.join(temporary, "prepared.wav")
                download_media(client, task_input, lease_token, original)
                stream_index = int(job.step.rsplit(".", 1)[1])
                prepare_audio(original, wav_path, stream_index)
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
                result = transcribe_prepared_audio(active_transcriber, wav_path, pipeline, vocabulary)
                result["audio_stream_index"] = stream_index
        elif is_voice_step(job.step):
            voice_value = getattr(task_input, "voice", None)
            voice_input = voice_value.to_dict() if hasattr(voice_value, "to_dict") else {}
            duration = getattr(task_input, "duration_seconds", None)
            duration = duration if isinstance(duration, (int, float)) else None
            voice_config = voice_input.get("config") or {}
            if requires_audio(voice_config):
                # Fail on a missing model path before downloading any media.
                ensure_provider_ready(voice_input, voice_providers)
                with tempfile.TemporaryDirectory(prefix="snitch-voice-") as temporary:
                    original = os.path.join(temporary, "original.media")
                    wav_path = os.path.join(temporary, "prepared.wav")
                    download_media(client, task_input, lease_token, original)
                    prepare_audio(original, wav_path, int(job.step.rsplit(".", 1)[1]))
                    result = process_voice_step(job.step, voice_input, duration, wav_path, voice_providers)
            else:
                result = process_voice_step(job.step, voice_input, duration, None, voice_providers)
        elif is_trial_step(job.step):
            voice_input = trial_voice_input(task_input)
            trial = voice_input.get("trial") or {}
            with tempfile.TemporaryDirectory(prefix="snitch-voice-") as temporary:
                original = os.path.join(temporary, "original.media")
                wav_path = os.path.join(temporary, "prepared.wav")
                download_media(client, task_input, lease_token, original)
                prepare_audio(original, wav_path, int(trial.get("audio_stream_index", 0)))
                result = process_trial_step(job.step, voice_input, wav_path, voice_providers)
        else:
            raise ValueError("unsupported worker step")
        heartbeat_stop.set()
        heartbeater.join()
        if lease.lost.is_set():
            # ADR 0029: a job whose lease was refused or expired sends no result.
            log.emit(logging.WARNING, "job_result_discarded", reason=lease.reason, **context)
            return True
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
    except (
        httpx.HTTPError,
        KeyError,
        OSError,
        RuntimeError,
        subprocess.SubprocessError,
        TypeError,
        ValueError,
    ) as exc:
        heartbeat_stop.set()
        heartbeater.join()
        if lease.lost.is_set():
            # The job was reassigned or expired: neither a result nor a failure may be recorded.
            log.emit(logging.WARNING, "job_result_discarded", reason=lease.reason, **context)
            return True
        diagnostic: dict[str, Any] = {}
        if isinstance(exc, TranscriptionError):
            error_code = exc.code
            diagnostic = {"provider": "openai-compatible", "duration_ms": int((time.monotonic() - started) * 1000)}
            model = getattr(transcriber, "model", None) or getattr(active_transcriber, "model", None)
            if isinstance(model, str) and model:
                diagnostic["model"] = model
            if exc.http_status is not None:
                diagnostic["http_status"] = exc.http_status
        elif isinstance(exc, VoiceProviderUnavailable) and _is_diarization_step(job.step) and exc.reason in DIARIZATION_FAILURE_CODES:
            error_code = DIARIZATION_FAILURE_CODES[exc.reason]
            voice_config = _voice_config(task_input)
            diagnostic = {"duration_ms": int((time.monotonic() - started) * 1000)}
            for field, key in (("provider", "provider"), ("model", "model"), ("revision", "model_version")):
                value = voice_config.get(key)
                if isinstance(value, str) and value:
                    diagnostic[field] = value
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
        except (httpx.HTTPError, RuntimeError):
            # The lease may have expired or moved while the provider was busy.
            # In that case the old attempt must not mutate the recovered task.
            log.emit(logging.WARNING, "job_failure_not_recorded", **context)
    return True


def _worker_loop(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float,
    heartbeat_interval: float,
    guard: revocation.RevocationGuard | None = None,
) -> None:
    while not stop.is_set():
        if guard is not None and guard.revoked.is_set():
            return
        try:
            completed = run_once(client, capacity, heartbeat_interval, stop)
        except WorkerUnauthorized:
            if guard is not None and guard.on_unauthorized(client):
                stop.set()
                return
            completed = False
        except (httpx.HTTPError, RuntimeError):
            # Network outages are retried on the next poll. Do not log exception
            # details because HTTP client errors can include sensitive URLs.
            completed = False
        if not completed:
            stop.wait(poll_interval)


def run_until_stopped(
    client: AuthenticatedClient,
    capacity: int,
    stop: threading.Event,
    poll_interval: float = 5.0,
    heartbeat_interval: float = 45.0,
    models_dir: str | None = None,
    presence_interval: float = PRESENCE_INTERVAL,
) -> bool:
    """Run a bounded pool; stop claiming on shutdown and drain active tasks.

    Presence runs on its own thread, so a long job never makes the Worker look Offline.
    Return True when Snitch confirmed this Worker's deletion (the models volume is then erased)."""
    if capacity < 1 or capacity > 4:
        raise ValueError("worker capacity must be an integer from 1 to 4")
    if poll_interval <= 0 or heartbeat_interval <= 0 or presence_interval <= 0:
        raise ValueError("worker intervals must be positive")

    guard = revocation.RevocationGuard(models_dir or revocation.models_dir(os.environ))
    presence = threading.Thread(
        target=_presence_loop,
        args=(client, capacity, stop, presence_interval, guard),
        daemon=True,
        name="snitch-presence",
    )
    presence.start()
    with ThreadPoolExecutor(max_workers=capacity) as executor:
        workers = [
            executor.submit(
                _worker_loop,
                client,
                capacity,
                stop,
                poll_interval,
                heartbeat_interval,
                guard,
            )
            for _ in range(capacity)
        ]
        while not stop.wait(0.1):
            pass
        for worker in workers:
            worker.result()
    presence.join()
    return guard.revoked.is_set()


def main() -> None:
    log.configure()
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
        revoked = run_until_stopped(client, capacity, stop, models_dir=models_dir)
    if revoked:
        revocation.wait_inert(shutdown)


if __name__ == "__main__":
    main()
