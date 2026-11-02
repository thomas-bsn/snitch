from enum import StrEnum

class WorkerVoiceModelStatusCauseCode(StrEnum):
    DISK_FULL = "disk_full"
    HF_ACCESS_DENIED = "hf_access_denied"
    HF_AUTH_REQUIRED = "hf_auth_required"
    HF_CACHE_ERROR = "hf_cache_error"
    HF_NETWORK = "hf_network"
    HF_NOT_FOUND = "hf_not_found"
    MODELS_VOLUME_UNWRITABLE = "models_volume_unwritable"
    HF_TOKEN_MISSING = "hf_token_missing"
    INFERENCE_FAILED = "inference_failed"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
