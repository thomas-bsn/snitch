from enum import StrEnum

class SourceHealth(StrEnum):
    READY = "ready"
    UNAVAILABLE = "unavailable"
    WARNING = "warning"

    def __str__(self) -> str:
        return str(self.value)
