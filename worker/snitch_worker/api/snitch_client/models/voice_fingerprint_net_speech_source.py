from enum import StrEnum

class VoiceFingerprintNetSpeechSource(StrEnum):
    ESTIMATED = "estimated"
    TURNS = "turns"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
