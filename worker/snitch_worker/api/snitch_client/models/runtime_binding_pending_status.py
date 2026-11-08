from enum import StrEnum

class RuntimeBindingPendingStatus(StrEnum):
    ACKNOWLEDGED = "acknowledged"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
