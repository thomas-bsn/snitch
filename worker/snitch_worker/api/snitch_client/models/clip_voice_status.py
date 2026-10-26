from enum import StrEnum

class ClipVoiceStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    NONE = "none"
    QUEUED = "queued"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
