from enum import StrEnum

class ClipAnalysisStepStep(StrEnum):
    PREPARATION = "preparation"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
