from enum import StrEnum

class VoiceModelWorkerStateState(StrEnum):
    DOWNLOADING = "downloading"
    FAILED = "failed"
    NOT_INSTALLED = "not_installed"
    READY = "ready"
    REQUESTED = "requested"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
