from enum import StrEnum

class SpeakerAttributionRuleRule(StrEnum):
    SEGMENT = "segment"
    WORD = "word"

    def __str__(self) -> str:
        return str(self.value)
