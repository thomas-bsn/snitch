from enum import StrEnum

class AudioPlanChildStatus(StrEnum):
    CANCELLED = "cancelled"
    DEAD = "dead"
    DONE = "done"
    QUEUED = "queued"
    RETRY = "retry"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
