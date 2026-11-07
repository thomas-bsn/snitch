from enum import StrEnum

class AudioAnalysisChoiceMode(StrEnum):
    FULL = "full"
    MISSING = "missing"
    STEPS = "steps"

    def __str__(self) -> str:
        return str(self.value)
