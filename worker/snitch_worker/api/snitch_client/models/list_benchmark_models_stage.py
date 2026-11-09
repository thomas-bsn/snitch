from enum import StrEnum

class ListBenchmarkModelsStage(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
