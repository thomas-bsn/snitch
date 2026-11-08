from enum import StrEnum

class EngineOrderIntent(StrEnum):
    INSTALL = "install"

    def __str__(self) -> str:
        return str(self.value)
