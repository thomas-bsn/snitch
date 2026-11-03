from enum import StrEnum

class WorkerTranscriptionFailureInputErrorType(StrEnum):
    CONNECTION = "connection"
    HTTP_STATUS = "http_status"
    INVALID_BODY = "invalid_body"
    TIMEOUT = "timeout"

    def __str__(self) -> str:
        return str(self.value)
