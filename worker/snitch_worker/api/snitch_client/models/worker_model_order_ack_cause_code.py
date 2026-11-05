from enum import StrEnum

class WorkerModelOrderAckCauseCode(StrEnum):
    INVALID_ORDER = "invalid_order"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
