from enum import StrEnum

class TaskCardStepState(StrEnum):
    CANCELLED = "cancelled"
    DEAD = "dead"
    DONE = "done"
    FAILED = "failed"
    NOT_APPLICABLE = "not_applicable"
    QUEUED = "queued"
    READY = "ready"
    RETRY = "retry"
    RUNNING = "running"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
