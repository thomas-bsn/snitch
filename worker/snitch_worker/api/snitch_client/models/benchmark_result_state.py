from enum import StrEnum

class BenchmarkResultState(StrEnum):
    COMPLETE = "complete"
    FAILED = "failed"
    PARTIAL = "partial"

    def __str__(self) -> str:
        return str(self.value)
