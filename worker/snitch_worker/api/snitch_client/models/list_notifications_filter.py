from enum import StrEnum

class ListNotificationsFilter(StrEnum):
    ACTIONS = "actions"
    ALL = "all"
    ERRORS = "errors"
    UNREAD = "unread"

    def __str__(self) -> str:
        return str(self.value)
