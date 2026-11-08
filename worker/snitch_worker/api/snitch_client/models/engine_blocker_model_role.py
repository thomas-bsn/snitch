from enum import StrEnum

class EngineBlockerModelRole(StrEnum):
    BOUND = "bound"
    PREVIOUS = "previous"

    def __str__(self) -> str:
        return str(self.value)
