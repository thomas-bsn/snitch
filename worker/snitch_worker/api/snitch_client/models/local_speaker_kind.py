from enum import StrEnum

class LocalSpeakerKind(StrEnum):
    DIARIZED = "diarized"
    MICROPHONE = "microphone"

    def __str__(self) -> str:
        return str(self.value)
