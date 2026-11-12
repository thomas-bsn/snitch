from enum import StrEnum

class TranscriptionSpeakerAttributionRule(StrEnum):
    SEGMENT = "segment"
    WORD = "word"

    def __str__(self) -> str:
        return str(self.value)
