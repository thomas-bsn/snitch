from enum import StrEnum

class WorkerTranscriptionFailureInputMethod(StrEnum):
    GET = "GET"

    def __str__(self) -> str:
        return str(self.value)
