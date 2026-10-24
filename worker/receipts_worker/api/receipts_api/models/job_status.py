from enum import StrEnum

class JobStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    LEASED = "leased"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
