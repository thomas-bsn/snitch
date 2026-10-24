from enum import StrEnum

class UploadStatus(StrEnum):
    CANCELLED = "cancelled"
    COMPLETE = "complete"
    EXPIRED = "expired"
    FAILED = "failed"
    PROCESSING = "processing"
    UPLOADING = "uploading"

    def __str__(self) -> str:
        return str(self.value)
