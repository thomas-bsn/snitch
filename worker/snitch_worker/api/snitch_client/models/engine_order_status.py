from enum import StrEnum

class EngineOrderStatus(StrEnum):
    ACKNOWLEDGED = "acknowledged"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
