from enum import StrEnum

class WorkerEngineOrderStatus(StrEnum):
    ACKNOWLEDGED = "acknowledged"
    CANCELLED = "cancelled"
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"
    REFUSED = "refused"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
