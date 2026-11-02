from enum import StrEnum

class ClipAnalysisRunForcedStepsItem(StrEnum):
    DIARIZATION = "diarization"
    INDEXING = "indexing"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
