from enum import StrEnum

class TranscriptionPipelinePreparation(StrEnum):
    WAV_16KHZ_MONO_V1 = "wav-16khz-mono-v1"

    def __str__(self) -> str:
        return str(self.value)
