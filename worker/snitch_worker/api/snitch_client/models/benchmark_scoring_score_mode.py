from enum import StrEnum

class BenchmarkScoringScoreMode(StrEnum):
    BEST_SAMPLE = "best_sample"
    CENTROID = "centroid"
    CUMULATIVE = "cumulative"

    def __str__(self) -> str:
        return str(self.value)
