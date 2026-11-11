from enum import StrEnum

class SpeakerAttributionRuleVersionRule(StrEnum):
    SEGMENT = "segment"
    WORD = "word"

    def __str__(self) -> str:
        return str(self.value)
