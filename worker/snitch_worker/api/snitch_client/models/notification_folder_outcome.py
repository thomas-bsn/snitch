from enum import StrEnum

class NotificationFolderOutcome(StrEnum):
    APPROVED = "approved"
    REFUSED = "refused"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
