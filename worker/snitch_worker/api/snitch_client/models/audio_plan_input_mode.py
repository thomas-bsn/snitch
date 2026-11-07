from enum import StrEnum

class AudioPlanInputMode(StrEnum):
    AUTO = "auto"
    MANUAL = "manual"

    def __str__(self) -> str:
        return str(self.value)
