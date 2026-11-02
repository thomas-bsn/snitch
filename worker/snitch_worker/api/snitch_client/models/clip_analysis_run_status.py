from enum import StrEnum

class ClipAnalysisRunStatus(StrEnum):
    ACTIVE = "active"
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    FAILED = "failed"
    INCOMPLETE = "incomplete"

    def __str__(self) -> str:
        return str(self.value)
