from enum import StrEnum

class AudioTrackDecisionAction(StrEnum):
    EXCLUDE = "exclude"
    INCLUDE = "include"

    def __str__(self) -> str:
        return str(self.value)
