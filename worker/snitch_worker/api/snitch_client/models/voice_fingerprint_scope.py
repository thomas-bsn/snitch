from enum import StrEnum

class VoiceFingerprintScope(StrEnum):
    SPEAKER = "speaker"
    TURN = "turn"

    def __str__(self) -> str:
        return str(self.value)
