from enum import StrEnum

class NotificationWebhookAuditItemAction(StrEnum):
    AUTO_DISABLED = "auto_disabled"
    CREATED = "created"
    DELETED = "deleted"
    ROTATED = "rotated"
    TESTED = "tested"
    UPDATED = "updated"

    def __str__(self) -> str:
        return str(self.value)
