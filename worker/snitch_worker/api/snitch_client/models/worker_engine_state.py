from enum import StrEnum

class WorkerEngineState(StrEnum):
    DOWNLOADING_RUNTIME = "downloading_runtime"
    FAILED = "failed"
    INSTALLING = "installing"
    READY = "ready"
    REMOVED = "removed"
    REMOVING = "removing"
    REQUESTED = "requested"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
