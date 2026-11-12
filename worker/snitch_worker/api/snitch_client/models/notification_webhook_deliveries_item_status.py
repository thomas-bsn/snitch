from enum import StrEnum

class NotificationWebhookDeliveriesItemStatus(StrEnum):
    FAILED = "failed"
    PENDING = "pending"
    SENT = "sent"
    SKIPPED = "skipped"

    def __str__(self) -> str:
        return str(self.value)
