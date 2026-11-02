from enum import StrEnum

class ClipAnalysisRunScope(StrEnum):
    FULL = "full"
    TRANSCRIPTION = "transcription"
    VOICE = "voice"

    def __str__(self) -> str:
        return str(self.value)
