from enum import StrEnum

class BenchmarkRunStage(StrEnum):
    DIARIZATION = "diarization"
    FINGERPRINT = "fingerprint"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
