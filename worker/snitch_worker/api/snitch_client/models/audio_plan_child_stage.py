from enum import StrEnum

class AudioPlanChildStage(StrEnum):
    DIAGNOSTIC = "diagnostic"
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
