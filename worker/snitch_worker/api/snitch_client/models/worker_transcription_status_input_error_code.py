from enum import StrEnum

class WorkerTranscriptionStatusInputErrorCode(StrEnum):
    CONNECTION_FAILED = "connection_failed"
    MODEL_MISSING = "model_missing"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
