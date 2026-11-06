from enum import StrEnum

class WorkerTranscriptionConfigErrorCode(StrEnum):
    CONNECTION_FAILED = "connection_failed"
    MODEL_MISSING = "model_missing"

    def __str__(self) -> str:
        return str(self.value)
