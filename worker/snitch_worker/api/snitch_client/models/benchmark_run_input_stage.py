from enum import StrEnum

class BenchmarkRunInputStage(StrEnum):
    FINGERPRINT = "fingerprint"

    def __str__(self) -> str:
        return str(self.value)
