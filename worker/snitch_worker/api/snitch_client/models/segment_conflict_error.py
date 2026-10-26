from enum import StrEnum

class SegmentConflictError(StrEnum):
    SEGMENT_CONFLICT = "segment_conflict"
    TRANSCRIPTION_CHANGED = "transcription_changed"

    def __str__(self) -> str:
        return str(self.value)
