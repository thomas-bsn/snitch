from enum import StrEnum

class VoiceGroupThresholdsScoreMode(StrEnum):
    BEST_SAMPLE = "best_sample"
    CUMULATIVE = "cumulative"

    def __str__(self) -> str:
        return str(self.value)
