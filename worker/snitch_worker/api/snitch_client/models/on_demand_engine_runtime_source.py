from enum import StrEnum

class OnDemandEngineRuntimeSource(StrEnum):
    ON_DEMAND = "on_demand"

    def __str__(self) -> str:
        return str(self.value)
