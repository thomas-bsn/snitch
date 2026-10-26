from enum import StrEnum

class SetupStepStateState(StrEnum):
    DONE = "done"
    SKIPPED = "skipped"
    TODO = "todo"

    def __str__(self) -> str:
        return str(self.value)
