from enum import StrEnum

class FailureCode(StrEnum):
    AUDIO_PREPARATION_FAILED = "audio_preparation_failed"
    CANCELLED_BY_ADMIN = "cancelled_by_admin"
    DIARIZATION_INFERENCE_FAILED = "diarization_inference_failed"
    GPU_OUT_OF_MEMORY = "gpu_out_of_memory"
    GPU_UNAVAILABLE = "gpu_unavailable"
    HF_AUTH_REQUIRED = "hf_auth_required"
    HF_CACHE_ERROR = "hf_cache_error"
    HF_NETWORK = "hf_network"
    HF_NOT_FOUND = "hf_not_found"
    INPUT_VERSION_SUPERSEDED = "input_version_superseded"
    INVALID_INPUT = "invalid_input"
    LEASE_EXPIRED = "lease_expired"
    MAX_ATTEMPTS_EXHAUSTED = "max_attempts_exhausted"
    MODELS_VOLUME_UNWRITABLE = "models_volume_unwritable"
    PROVIDER_UNAVAILABLE = "provider_unavailable"
    PYANNOTE_ACCESS_DENIED = "pyannote_access_denied"
    PYANNOTE_MODEL_MISSING = "pyannote_model_missing"
    WHISPER_HTTP_ERROR = "whisper_http_error"
    WHISPER_INVALID_RESPONSE = "whisper_invalid_response"
    WHISPER_MODEL_MISSING = "whisper_model_missing"
    WHISPER_NETWORK = "whisper_network"
    WHISPER_TIMEOUT = "whisper_timeout"
    WORKER_FAILED = "worker_failed"
    WORKER_REVOKED = "worker_revoked"

    def __str__(self) -> str:
        return str(self.value)
