from enum import StrEnum

class WorkerEngineSource(StrEnum):
    ON_DEMAND = "on_demand"
    SIGNED = "signed"

    def __str__(self) -> str:
        return str(self.value)
