from enum import StrEnum

class PostWorkerUpdateStatusBodyState(StrEnum):
    FAILED = "failed"
    IDLE = "idle"
    PULLING = "pulling"
    RESTARTING = "restarting"
    ROLLED_BACK = "rolled_back"
    UPDATED = "updated"

    def __str__(self) -> str:
        return str(self.value)
