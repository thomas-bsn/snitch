from enum import StrEnum

class ClipAnalysisRunMode(StrEnum):
    FULL = "full"
    MISSING = "missing"
    STEPS = "steps"

    def __str__(self) -> str:
        return str(self.value)
