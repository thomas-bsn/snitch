from enum import StrEnum

class BenchmarkRunInputStage(StrEnum):
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
