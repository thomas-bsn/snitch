from enum import StrEnum

class ListBenchmarkModelsStage(StrEnum):
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
