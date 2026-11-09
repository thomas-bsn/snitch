from enum import StrEnum

class ListBenchmarkModelsStage(StrEnum):
    FINGERPRINT = "fingerprint"

    def __str__(self) -> str:
        return str(self.value)
