from enum import StrEnum

class ListLibraryExplorerVisibility(StrEnum):
    PRIVATE = "private"
    SHARED = "shared"

    def __str__(self) -> str:
        return str(self.value)
