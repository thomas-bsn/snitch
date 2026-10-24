from enum import StrEnum

class AudioTrackSelectionStatus(StrEnum):
    SELECTED = "selected"
    SELECTION_REQUIRED = "selection_required"

    def __str__(self) -> str:
        return str(self.value)
