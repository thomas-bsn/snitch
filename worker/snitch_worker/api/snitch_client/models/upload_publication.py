from enum import StrEnum

class UploadPublication(StrEnum):
    FAILED = "failed"
    PENDING = "pending"
    PUBLISHED = "published"

    def __str__(self) -> str:
        return str(self.value)
