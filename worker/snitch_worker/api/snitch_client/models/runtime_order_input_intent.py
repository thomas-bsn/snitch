from enum import StrEnum

class RuntimeOrderInputIntent(StrEnum):
    ROLLBACK = "rollback"
    UPDATE = "update"

    def __str__(self) -> str:
        return str(self.value)
