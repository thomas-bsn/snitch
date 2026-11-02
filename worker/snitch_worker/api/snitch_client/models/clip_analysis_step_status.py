from enum import StrEnum

class ClipAnalysisStepStatus(StrEnum):
    BLOCKED = "blocked"
    DONE = "done"
    FAILED = "failed"
    NOT_APPLICABLE = "not_applicable"
    NOT_YET_SUPPORTED = "not_yet_supported"
    RETRY = "retry"
    RUNNING = "running"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
