from enum import StrEnum

class WorkerVoiceModelRole(StrEnum):
    ACTIVE = "active"
    CANDIDATE = "candidate"
    REMOVAL = "removal"

    def __str__(self) -> str:
        return str(self.value)
