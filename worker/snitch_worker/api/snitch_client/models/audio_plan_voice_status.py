from enum import StrEnum

class AudioPlanVoiceStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    NOT_PUBLISHED = "not_published"
    NOT_STARTED = "not_started"
    QUEUED = "queued"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
