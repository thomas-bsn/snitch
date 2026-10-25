from enum import StrEnum

class MicrophoneTrackEventAction(StrEnum):
    DISABLE = "disable"
    REMOVE = "remove"
    SET = "set"

    def __str__(self) -> str:
        return str(self.value)
