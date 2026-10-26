from enum import StrEnum

class ClipStepState(StrEnum):
    DONE = "done"
    FAILED = "failed"
    NONE = "none"
    QUEUED = "queued"
    RUNNING = "running"
    UNAVAILABLE = "unavailable"

    def __str__(self) -> str:
        return str(self.value)
