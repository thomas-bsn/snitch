from enum import StrEnum

class WorkerEngineHistoryEntryState(StrEnum):
    DOWNLOADING_RUNTIME = "downloading_runtime"
    FAILED = "failed"
    INSTALLING = "installing"
    READY = "ready"
    REMOVED = "removed"
    REMOVING = "removing"
    REQUESTED = "requested"
    UNINSTALL_FAILED = "uninstall_failed"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
