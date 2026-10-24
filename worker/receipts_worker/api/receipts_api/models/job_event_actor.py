from enum import StrEnum

class JobEventActor(StrEnum):
    ADMIN = "admin"
    SYSTEM = "system"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
