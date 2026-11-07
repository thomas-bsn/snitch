from enum import StrEnum

class AudioSelectionDiagnosticFreshness(StrEnum):
    CURRENT = "current"
    MISSING = "missing"
    STALE = "stale"

    def __str__(self) -> str:
        return str(self.value)
