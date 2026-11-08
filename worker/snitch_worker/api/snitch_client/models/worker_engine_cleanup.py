from enum import StrEnum

class WorkerEngineCleanup(StrEnum):
    DONE = "done"
    REQUESTED = "requested"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
