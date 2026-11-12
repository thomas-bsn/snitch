from enum import StrEnum

class SpeakerAttributionRuleInputRule(StrEnum):
    SEGMENT = "segment"
    WORD = "word"

    def __str__(self) -> str:
        return str(self.value)
