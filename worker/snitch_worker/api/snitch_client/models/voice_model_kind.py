from enum import StrEnum

class VoiceModelKind(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    SEPARATION = "separation"

    def __str__(self) -> str:
        return str(self.value)
