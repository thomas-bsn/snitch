from enum import StrEnum

class PersonReferenceLevel(StrEnum):
    GOOD = "good"
    NONE = "none"
    WEAK = "weak"

    def __str__(self) -> str:
        return str(self.value)
