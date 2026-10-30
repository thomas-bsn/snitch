from enum import StrEnum

class ListJobsWorkflow(StrEnum):
    ACTIVE = "active"
    DONE = "done"

    def __str__(self) -> str:
        return str(self.value)
