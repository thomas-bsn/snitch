from enum import StrEnum

class PreviewBenchmarkActivationStage(StrEnum):
    DIARIZATION = "diarization"
    TRANSCRIPTION = "transcription"

    def __str__(self) -> str:
        return str(self.value)
