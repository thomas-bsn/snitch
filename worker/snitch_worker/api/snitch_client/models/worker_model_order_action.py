from enum import StrEnum

class WorkerModelOrderAction(StrEnum):
    INSTALL = "install"

    def __str__(self) -> str:
        return str(self.value)
