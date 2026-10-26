from enum import StrEnum

class VoiceVerificationCaseDecisionLevel(StrEnum):
    AMBIGUOUS = "ambiguous"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
