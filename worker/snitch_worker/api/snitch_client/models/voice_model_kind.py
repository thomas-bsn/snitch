from enum import StrEnum

class VoiceModelKind(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"

    def __str__(self) -> str:
        return str(self.value)
