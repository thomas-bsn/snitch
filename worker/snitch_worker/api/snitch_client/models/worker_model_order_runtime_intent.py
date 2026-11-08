from enum import StrEnum

class WorkerModelOrderRuntimeIntent(StrEnum):
    INSTALL = "install"
    ROLLBACK = "rollback"
    UPDATE = "update"

    def __str__(self) -> str:
        return str(self.value)
