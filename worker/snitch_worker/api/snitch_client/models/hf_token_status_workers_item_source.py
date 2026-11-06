from enum import StrEnum

class HFTokenStatusWorkersItemSource(StrEnum):
    CENTRAL = "central"
    LOCAL = "local"
    NONE = "none"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
