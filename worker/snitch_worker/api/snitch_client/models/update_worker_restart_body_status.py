from enum import StrEnum

class UpdateWorkerRestartBodyStatus(StrEnum):
    FAILED = "failed"
    ONLINE = "online"
    STARTING = "starting"
    STOPPING = "stopping"

    def __str__(self) -> str:
        return str(self.value)
