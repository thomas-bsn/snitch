from enum import StrEnum

class TranscriptRevisionTimeBase(StrEnum):
    CLIP = "clip"

    def __str__(self) -> str:
        return str(self.value)
