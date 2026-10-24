from enum import StrEnum

class TranscriptionPipelineProvider(StrEnum):
    OPENAI_COMPATIBLE = "openai-compatible"

    def __str__(self) -> str:
        return str(self.value)
