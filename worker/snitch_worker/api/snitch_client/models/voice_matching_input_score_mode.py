from enum import StrEnum

class VoiceMatchingInputScoreMode(StrEnum):
    BEST_SAMPLE = "best_sample"
    CUMULATIVE = "cumulative"

    def __str__(self) -> str:
        return str(self.value)
