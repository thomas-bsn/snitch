from enum import StrEnum

class WorkerInstallEventStatus(StrEnum):
    FAILED = "failed"
    STARTED = "started"
    SUCCEEDED = "succeeded"

    def __str__(self) -> str:
        return str(self.value)
