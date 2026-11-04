from enum import StrEnum

class StartLibraryIndexRunBodyMode(StrEnum):
    CARDS = "cards"
    FULL = "full"

    def __str__(self) -> str:
        return str(self.value)
