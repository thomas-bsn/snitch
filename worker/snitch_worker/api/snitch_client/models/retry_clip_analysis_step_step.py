from enum import StrEnum

class RetryClipAnalysisStepStep(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINTS = "fingerprints"
    INDEXING = "indexing"
    MATCHING = "matching"
    PREPARATION = "preparation"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
