from enum import StrEnum

class WorkerModelCellReplacementPhase(StrEnum):
    PREPARING = "preparing"
    REMOVING = "removing"
    SWITCHING = "switching"

    def __str__(self) -> str:
        return str(self.value)
