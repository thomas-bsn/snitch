from enum import StrEnum

class WorkerInstallPhase(StrEnum):
    GPU = "gpu"
    IMAGE = "image"
    PERMISSIONS = "permissions"
    PREREQUISITES = "prerequisites"
    REGISTRATION = "registration"
    SERVICE = "service"
    VOICE_MODELS = "voice-models"
    VOICE_TEST = "voice-test"

    def __str__(self) -> str:
        return str(self.value)
