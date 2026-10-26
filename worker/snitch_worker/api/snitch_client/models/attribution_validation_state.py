from enum import StrEnum

class AttributionValidationState(StrEnum):
    UNVALIDATED = "unvalidated"
    VALIDATED = "validated"

    def __str__(self) -> str:
        return str(self.value)
