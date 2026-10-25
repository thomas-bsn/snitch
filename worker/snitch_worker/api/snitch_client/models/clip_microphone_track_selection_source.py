from enum import StrEnum

class ClipMicrophoneTrackSelectionSource(StrEnum):
    CLIP_OVERRIDE = "clip_override"
    SOURCE_RULE = "source_rule"

    def __str__(self) -> str:
        return str(self.value)
