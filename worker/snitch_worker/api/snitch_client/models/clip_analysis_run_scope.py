from enum import StrEnum

class ClipAnalysisRunScope(StrEnum):
    PREPARATION = "preparation"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
