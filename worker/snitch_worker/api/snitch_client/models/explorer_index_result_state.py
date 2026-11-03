from enum import StrEnum

class ExplorerIndexResultState(StrEnum):
    INDEXED = "indexed"
    IN_PROGRESS = "in_progress"

    def __str__(self) -> str:
        return str(self.value)
