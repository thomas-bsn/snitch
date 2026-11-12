from enum import StrEnum

class NotificationAccountOutcome(StrEnum):
    APPROVED = "approved"
    REFUSED = "refused"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
