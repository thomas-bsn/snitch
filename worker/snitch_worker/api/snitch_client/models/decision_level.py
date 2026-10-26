from enum import StrEnum

class DecisionLevel(StrEnum):
    AMBIGUOUS = "ambiguous"
    AUTOMATIC = "automatic"
    HUMAN = "human"
    SUGGESTION = "suggestion"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
