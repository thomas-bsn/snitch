from enum import StrEnum

class PersonReferenceDiscardReason(StrEnum):
    CLIPPED = "clipped"
    LOW_QUALITY = "low_quality"
    NO_FINGERPRINT = "no_fingerprint"
    OVERLAP_ONLY = "overlap_only"
    TOO_SHORT = "too_short"

    def __str__(self) -> str:
        return str(self.value)
