from enum import StrEnum

class ClipAnalysisStepStatus(StrEnum):
    BLOCKED = "blocked"
    DONE = "done"
    FAILED = "failed"
    NOT_APPLICABLE = "not_applicable"
    RETRY = "retry"
    RUNNING = "running"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
