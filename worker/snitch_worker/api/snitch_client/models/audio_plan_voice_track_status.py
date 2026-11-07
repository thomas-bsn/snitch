from enum import StrEnum

class AudioPlanVoiceTrackStatus(StrEnum):
    CANCELLED = "cancelled"
    DONE = "done"
    FAILED = "failed"
    NOT_APPLICABLE = "not_applicable"
    NOT_STARTED = "not_started"
    QUEUED = "queued"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
