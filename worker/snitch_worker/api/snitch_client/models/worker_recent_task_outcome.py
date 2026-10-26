from enum import StrEnum

class WorkerRecentTaskOutcome(StrEnum):
    COMPLETED = "completed"
    FAILED = "failed"
    RETRY = "retry"
    STARTED = "started"

    def __str__(self) -> str:
        return str(self.value)
