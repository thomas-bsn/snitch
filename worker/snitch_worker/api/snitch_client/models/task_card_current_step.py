from enum import StrEnum

class TaskCardCurrentStep(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    RECOGNITION = "recognition"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
