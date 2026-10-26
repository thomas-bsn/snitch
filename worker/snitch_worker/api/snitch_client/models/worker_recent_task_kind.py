from enum import StrEnum

class WorkerRecentTaskKind(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    OTHER = "other"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
