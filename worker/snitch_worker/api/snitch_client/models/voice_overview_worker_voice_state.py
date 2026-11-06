from enum import StrEnum

class VoiceOverviewWorkerVoiceState(StrEnum):
    MODELS_MISSING = "models_missing"
    READY = "ready"
    UNAVAILABLE = "unavailable"
    UNTESTED = "untested"

    def __str__(self) -> str:
        return str(self.value)
