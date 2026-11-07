from enum import StrEnum

class WorkerRestartCommandStatus(StrEnum):
    ACCEPTED = "accepted"
    EXPIRED = "expired"
    FAILED = "failed"
    ONLINE = "online"
    REQUESTED = "requested"
    STARTING = "starting"
    STOPPING = "stopping"

    def __str__(self) -> str:
        return str(self.value)
