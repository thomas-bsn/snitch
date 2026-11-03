from enum import StrEnum

class ListLibraryExplorerVoices(StrEnum):
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
