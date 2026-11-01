from enum import StrEnum

class WorkerVoiceModelStatusState(StrEnum):
    DOWNLOADING = "downloading"
    FAILED = "failed"
    READY = "ready"
    REMOVED = "removed"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
