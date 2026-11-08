from enum import StrEnum

class RuntimeBindingHistoryEntryState(StrEnum):
    DOWNLOADING = "downloading"
    FAILED = "failed"
    PUBLISHED = "published"
    REQUESTED = "requested"
    VERIFYING = "verifying"

    def __str__(self) -> str:
        return str(self.value)
