from enum import StrEnum

class JobEventEvent(StrEnum):
    ADMIN_RELAUNCHED = "admin_relaunched"
    ATTEMPT_STARTED = "attempt_started"
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    LEASE_EXPIRED = "lease_expired"
    MARKED_DEAD = "marked_dead"
    RETRY_SCHEDULED = "retry_scheduled"
    WORKER_REVOKED = "worker_revoked"

    def __str__(self) -> str:
        return str(self.value)
