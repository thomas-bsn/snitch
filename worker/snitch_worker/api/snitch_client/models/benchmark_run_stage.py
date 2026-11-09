from enum import StrEnum

class BenchmarkRunStage(StrEnum):
    FINGERPRINT = "fingerprint"

    def __str__(self) -> str:
        return str(self.value)
