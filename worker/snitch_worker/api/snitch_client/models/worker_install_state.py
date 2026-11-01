from enum import StrEnum

class WorkerInstallState(StrEnum):
    CONNECTED = "connected"
    FAILED = "failed"
    INSTALLING = "installing"
    NEVER_CONNECTED = "never_connected"
    VOICE_READY = "voice_ready"

    def __str__(self) -> str:
        return str(self.value)
