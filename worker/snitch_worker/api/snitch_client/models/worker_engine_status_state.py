from enum import StrEnum

class WorkerEngineStatusState(StrEnum):
    DOWNLOADING_RUNTIME = "downloading_runtime"
    FAILED = "failed"
    INSTALLING = "installing"
    READY = "ready"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
