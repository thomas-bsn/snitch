from enum import StrEnum

class VoiceModelOrderPreviewState(StrEnum):
    IN_PROGRESS = "in_progress"
    MISSING = "missing"
    READY = "ready"

    def __str__(self) -> str:
        return str(self.value)
