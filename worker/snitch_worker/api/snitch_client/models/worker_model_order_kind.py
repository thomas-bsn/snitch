from enum import StrEnum

class WorkerModelOrderKind(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    SEPARATION = "separation"

    def __str__(self) -> str:
        return str(self.value)
