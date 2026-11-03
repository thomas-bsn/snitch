from enum import StrEnum

class WorkerHFTokenAckOutcome(StrEnum):
    APPLIED = "applied"
    FAILED = "failed"
    KEPT_LOCAL = "kept_local"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
