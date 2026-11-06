from enum import StrEnum

class ListLibraryExplorerRecursive(StrEnum):
    TRUE = "true"
    VALUE_0 = "1"

    def __str__(self) -> str:
        return str(self.value)
