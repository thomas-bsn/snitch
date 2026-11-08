from enum import StrEnum

class RuntimeBindingHistoryEntryIntent(StrEnum):
    ROLLBACK = "rollback"
    UPDATE = "update"

    def __str__(self) -> str:
        return str(self.value)
