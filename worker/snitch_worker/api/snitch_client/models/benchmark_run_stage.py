from enum import StrEnum

class BenchmarkRunStage(StrEnum):
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
