from enum import StrEnum

class VoiceTraceFingerprintUnusableReason(StrEnum):
    CLIPPED = "clipped"
    LOW_QUALITY = "low_quality"
    OVERLAP_ONLY = "overlap_only"
    TOO_SHORT = "too_short"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
