from enum import StrEnum

class WorkerModelOrderStatus(StrEnum):
    ACKNOWLEDGED = "acknowledged"
    CANCELLED = "cancelled"
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"
    REFUSED = "refused"

    def __str__(self) -> str:
        return str(self.value)
