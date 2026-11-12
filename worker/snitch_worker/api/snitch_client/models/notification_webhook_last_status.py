from enum import StrEnum

class NotificationWebhookLastStatus(StrEnum):
    FAILED = "failed"
    NEVER = "never"
    RATE_LIMITED = "rate_limited"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)
