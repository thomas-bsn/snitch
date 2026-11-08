from enum import StrEnum

class WorkerVoiceInputFingerprintModelEngine(StrEnum):
    PYANNOTE = "pyannote"

    def __str__(self) -> str:
        return str(self.value)
