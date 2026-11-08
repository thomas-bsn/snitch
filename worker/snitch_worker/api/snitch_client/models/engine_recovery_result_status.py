from enum import StrEnum

class EngineRecoveryResultStatus(StrEnum):
    CANCELLED = "cancelled"
    CANCELLING = "cancelling"
    DONE = "done"
    REQUESTED = "requested"

    def __str__(self) -> str:
        return str(self.value)
