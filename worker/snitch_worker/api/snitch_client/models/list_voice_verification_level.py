from enum import StrEnum

class ListVoiceVerificationLevel(StrEnum):
    ALL = "all"
    AMBIGUOUS = "ambiguous"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
