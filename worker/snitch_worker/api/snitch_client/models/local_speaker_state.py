from enum import StrEnum

class LocalSpeakerState(StrEnum):
    UNIDENTIFIED = "unidentified"

    def __str__(self) -> str:
        return str(self.value)
