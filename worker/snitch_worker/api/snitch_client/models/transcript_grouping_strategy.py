from enum import StrEnum

class TranscriptGroupingStrategy(StrEnum):
    EXACT_PCM = "exact_pcm"
    GAIN = "gain"

    def __str__(self) -> str:
        return str(self.value)
