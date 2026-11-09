from enum import StrEnum

class BenchmarkDatasetStagesItem(StrEnum):
    DIARIZATION = "diarization"
    E2E = "e2e"
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
