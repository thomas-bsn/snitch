from enum import StrEnum

class SourceFileStatus(StrEnum):
    MISSING = "missing"
    READY = "ready"
    UNREADABLE = "unreadable"
    WAITING_STABILITY = "waiting_stability"

    def __str__(self) -> str:
        return str(self.value)
