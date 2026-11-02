from enum import StrEnum

class NotificationKind(StrEnum):
    JOB_FAILED = "job_failed"
    MODEL_MISSING = "model_missing"
    NEEDS_SPEAKER_REVIEW = "needs_speaker_review"
    PROVIDER_UNAVAILABLE = "provider_unavailable"
    TECHNICAL_INCIDENT = "technical_incident"
    WORKER_UNAVAILABLE = "worker_unavailable"

    def __str__(self) -> str:
        return str(self.value)
