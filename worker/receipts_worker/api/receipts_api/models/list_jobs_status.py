from enum import StrEnum

class ListJobsStatus(StrEnum):
    CANCELLED = "cancelled"
    DEAD = "dead"
    DONE = "done"
    FAILED = "failed"
    LEASED = "leased"
    PENDING = "pending"
    RETRY_WAIT = "retry_wait"

    def __str__(self) -> str:
        return str(self.value)
