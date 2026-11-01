from enum import StrEnum

class WorkerInstallEventInputStatus(StrEnum):
    FAILED = "failed"
    STARTED = "started"
    SUCCEEDED = "succeeded"

    def __str__(self) -> str:
        return str(self.value)
