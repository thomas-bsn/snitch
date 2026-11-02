from enum import StrEnum

class ListNotificationsState(StrEnum):
    ACTIVE = "active"
    ALL = "all"
    RESOLVED = "resolved"

    def __str__(self) -> str:
        return str(self.value)
