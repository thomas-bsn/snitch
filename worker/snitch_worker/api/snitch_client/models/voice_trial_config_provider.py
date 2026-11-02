from enum import StrEnum

class VoiceTrialConfigProvider(StrEnum):
    NEMO = "nemo"
    PYANNOTE = "pyannote"

    def __str__(self) -> str:
        return str(self.value)
