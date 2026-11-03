from enum import StrEnum

class GetWorkerUpdatesResponse200WorkersItemState(StrEnum):
    FAILED = "failed"
    IDLE = "idle"
    PULLING = "pulling"
    RESTARTING = "restarting"
    ROLLED_BACK = "rolled_back"
    UPDATED = "updated"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
