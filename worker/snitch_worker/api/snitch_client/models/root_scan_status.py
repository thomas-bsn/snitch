from enum import StrEnum

class RootScanStatus(StrEnum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
