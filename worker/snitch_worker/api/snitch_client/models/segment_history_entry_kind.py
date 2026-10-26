from enum import StrEnum

class SegmentHistoryEntryKind(StrEnum):
    RESTORE = "restore"
    SPEAKER = "speaker"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
