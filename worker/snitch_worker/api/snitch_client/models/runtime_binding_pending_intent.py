from enum import StrEnum

class RuntimeBindingPendingIntent(StrEnum):
    ROLLBACK = "rollback"
    UPDATE = "update"

    def __str__(self) -> str:
        return str(self.value)
