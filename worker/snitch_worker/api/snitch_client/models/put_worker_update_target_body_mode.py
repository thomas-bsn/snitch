from enum import StrEnum

class PutWorkerUpdateTargetBodyMode(StrEnum):
    DISABLED = "disabled"
    LATEST = "latest"
    PINNED = "pinned"

    def __str__(self) -> str:
        return str(self.value)
