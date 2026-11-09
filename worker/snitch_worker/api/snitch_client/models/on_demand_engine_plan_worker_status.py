from enum import StrEnum

class OnDemandEnginePlanWorkerStatus(StrEnum):
    OK = "ok"
    REFUSED = "refused"

    def __str__(self) -> str:
        return str(self.value)
