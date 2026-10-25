from enum import StrEnum

class MicrophoneTrackEventScope(StrEnum):
    CLIP = "clip"
    SOURCE = "source"

    def __str__(self) -> str:
        return str(self.value)
