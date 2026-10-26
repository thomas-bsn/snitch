from enum import StrEnum

class SetupStepInputState(StrEnum):
    DONE = "done"
    SKIPPED = "skipped"

    def __str__(self) -> str:
        return str(self.value)
