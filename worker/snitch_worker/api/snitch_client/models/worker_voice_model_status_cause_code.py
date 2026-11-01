from enum import StrEnum

class WorkerVoiceModelStatusCauseCode(StrEnum):
    DISK_FULL = "disk_full"
    HF_ACCESS_DENIED = "hf_access_denied"
    HF_TOKEN_MISSING = "hf_token_missing"
    INFERENCE_FAILED = "inference_failed"
    RUNTIME_INCOMPATIBLE = "runtime_incompatible"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
