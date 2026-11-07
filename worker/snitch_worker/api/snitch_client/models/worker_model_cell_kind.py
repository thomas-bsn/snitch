from enum import StrEnum

class WorkerModelCellKind(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    SEPARATION = "separation"
    VAD = "vad"

    def __str__(self) -> str:
        return str(self.value)
