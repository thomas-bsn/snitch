from enum import StrEnum

class VoiceEngineManifestCapabilitiesItem(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"

    def __str__(self) -> str:
        return str(self.value)
