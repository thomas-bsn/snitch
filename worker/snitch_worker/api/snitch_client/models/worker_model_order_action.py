from enum import StrEnum

class WorkerModelOrderAction(StrEnum):
    INSTALL = "install"
    REMOVE = "remove"

    def __str__(self) -> str:
        return str(self.value)
