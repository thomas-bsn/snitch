from enum import StrEnum

class AudioAnalysisChoiceScope(StrEnum):
    FULL = "full"
    TRANSCRIPTION = "transcription"
    VOICE = "voice"

    def __str__(self) -> str:
        return str(self.value)
