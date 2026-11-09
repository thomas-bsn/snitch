from enum import StrEnum

class BenchmarkActivationInputStage(StrEnum):
    DIARIZATION = "diarization"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
