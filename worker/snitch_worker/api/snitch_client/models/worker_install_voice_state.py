from enum import StrEnum

class WorkerInstallVoiceState(StrEnum):
    PENDING = "pending"
    READY = "ready"
    UNAVAILABLE = "unavailable"

    def __str__(self) -> str:
        return str(self.value)
