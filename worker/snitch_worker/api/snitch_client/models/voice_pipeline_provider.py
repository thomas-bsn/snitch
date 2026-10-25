from enum import StrEnum

class VoicePipelineProvider(StrEnum):
    FAKE = "fake"
    PYANNOTE = "pyannote"

    def __str__(self) -> str:
        return str(self.value)
