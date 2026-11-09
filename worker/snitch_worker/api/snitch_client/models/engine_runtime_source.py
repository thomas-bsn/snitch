from enum import StrEnum

class EngineRuntimeSource(StrEnum):
    ON_DEMAND = "on_demand"

    def __str__(self) -> str:
        return str(self.value)
