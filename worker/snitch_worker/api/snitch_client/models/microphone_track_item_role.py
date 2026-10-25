from enum import StrEnum

class MicrophoneTrackItemRole(StrEnum):
    CANDIDATE = "candidate"
    MICROPHONE = "microphone"
    OTHER = "other"

    def __str__(self) -> str:
        return str(self.value)
