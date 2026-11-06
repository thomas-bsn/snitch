from enum import StrEnum

class IndexRunStatus(StrEnum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    PAUSED = "paused"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
