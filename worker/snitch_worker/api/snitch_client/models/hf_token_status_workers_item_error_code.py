from enum import StrEnum

class HFTokenStatusWorkersItemErrorCode(StrEnum):
    HF_TOKEN_STORE_FAILED = "hf_token_store_failed"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
