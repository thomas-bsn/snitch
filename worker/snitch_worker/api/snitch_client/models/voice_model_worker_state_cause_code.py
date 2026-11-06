from enum import StrEnum

class VoiceModelWorkerStateCauseCode(StrEnum):
    DISK_FULL = "disk_full"
    HF_ACCESS_DENIED = "hf_access_denied"
    HF_AUTH_REQUIRED = "hf_auth_required"
    HF_CACHE_ERROR = "hf_cache_error"
    HF_GATED_DENIED = "hf_gated_denied"
    HF_NETWORK = "hf_network"
    HF_NOT_FOUND = "hf_not_found"
    HF_REPO_INVALID = "hf_repo_invalid"
    HF_TOKEN_INVALID = "hf_token_invalid"
    HF_TOKEN_MISSING = "hf_token_missing"
    INFERENCE_FAILED = "inference_failed"
    INVALID_ORDER = "invalid_order"
    MODELS_VOLUME_UNWRITABLE = "models_volume_unwritable"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
