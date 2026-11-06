from enum import StrEnum

class FingerprintTransitionPhase(StrEnum):
    CANCELLED = "cancelled"
    PREPARING = "preparing"
    RECOMPUTING = "recomputing"
    SWITCHED = "switched"

    def __str__(self) -> str:
        return str(self.value)
