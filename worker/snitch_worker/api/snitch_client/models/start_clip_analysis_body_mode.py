from enum import StrEnum

class StartClipAnalysisBodyMode(StrEnum):
    FULL = "full"
    MISSING = "missing"
    STEPS = "steps"

    def __str__(self) -> str:
        return str(self.value)
