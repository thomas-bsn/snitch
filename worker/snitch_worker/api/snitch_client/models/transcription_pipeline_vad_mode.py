from enum import StrEnum

class TranscriptionPipelineVadMode(StrEnum):
    ENERGY = "energy"
    OFF = "off"

    def __str__(self) -> str:
        return str(self.value)
