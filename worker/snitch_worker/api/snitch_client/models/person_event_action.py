from enum import StrEnum

class PersonEventAction(StrEnum):
    DELETE = "delete"
    MERGE = "merge"
    SPLIT = "split"

    def __str__(self) -> str:
        return str(self.value)
