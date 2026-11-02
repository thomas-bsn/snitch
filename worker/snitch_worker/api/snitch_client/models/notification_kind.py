from enum import StrEnum

class NotificationKind(StrEnum):
    NEEDS_SPEAKER_REVIEW = "needs_speaker_review"

    def __str__(self) -> str:
        return str(self.value)
