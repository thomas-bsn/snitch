from enum import StrEnum

class TaskCardCurrentStep(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    INDEXING = "indexing"
    RECOGNITION = "recognition"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
