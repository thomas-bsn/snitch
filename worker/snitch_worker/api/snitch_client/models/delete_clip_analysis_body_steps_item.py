from enum import StrEnum

class DeleteClipAnalysisBodyStepsItem(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINTS = "fingerprints"
    MATCHING = "matching"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
