from enum import StrEnum

class ListLibraryExplorerStage(StrEnum):
    COMPLETE = "complete"
    INDEXED = "indexed"
    TRANSCRIBED = "transcribed"
    VOICES = "voices"

    def __str__(self) -> str:
        return str(self.value)
