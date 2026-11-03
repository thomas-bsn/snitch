from enum import StrEnum

class WorkerHFTokenAckSource(StrEnum):
    CENTRAL = "central"
    LOCAL = "local"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
