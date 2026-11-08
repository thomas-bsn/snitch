from enum import StrEnum

class WorkerVoiceInputFingerprintModelEngine(StrEnum):
    NEMO_TITANET = "nemo-titanet"
    PYANNOTE = "pyannote"
    SPEECHBRAIN_ECAPA = "speechbrain-ecapa"

    def __str__(self) -> str:
        return str(self.value)
