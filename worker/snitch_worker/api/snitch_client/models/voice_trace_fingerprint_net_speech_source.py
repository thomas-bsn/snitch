from enum import StrEnum

class VoiceTraceFingerprintNetSpeechSource(StrEnum):
    ESTIMATED = "estimated"
    TURNS = "turns"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
