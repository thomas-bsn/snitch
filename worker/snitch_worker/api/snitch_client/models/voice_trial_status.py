from enum import StrEnum

class VoiceTrialStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
