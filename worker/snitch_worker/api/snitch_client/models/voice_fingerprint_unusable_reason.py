from enum import StrEnum

class VoiceFingerprintUnusableReason(StrEnum):
    LOW_QUALITY = "low_quality"
    TOO_SHORT = "too_short"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
