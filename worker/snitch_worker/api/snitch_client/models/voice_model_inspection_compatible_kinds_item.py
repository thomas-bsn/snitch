from enum import StrEnum

class VoiceModelInspectionCompatibleKindsItem(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    SEPARATION = "separation"
    VAD = "vad"

    def __str__(self) -> str:
        return str(self.value)
