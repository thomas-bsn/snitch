from enum import StrEnum

class WorkerPageTranscriptionMode(StrEnum):
    DISABLED = "disabled"
    ENABLED = "enabled"
    UNCONFIGURED = "unconfigured"

    def __str__(self) -> str:
        return str(self.value)
