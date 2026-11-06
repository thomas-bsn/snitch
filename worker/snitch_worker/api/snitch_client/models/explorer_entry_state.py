from enum import StrEnum

class ExplorerEntryState(StrEnum):
    ANALYZED = "analyzed"
    FAILED = "failed"
    IN_PROGRESS = "in_progress"
    NOT_ANALYZED = "not_analyzed"
    NOT_INDEXED = "not_indexed"
    TRANSCRIBED = "transcribed"

    def __str__(self) -> str:
        return str(self.value)
