from enum import StrEnum

class VoiceOverviewProvider(StrEnum):
    FAKE = "fake"
    PYANNOTE = "pyannote"

    def __str__(self) -> str:
        return str(self.value)
