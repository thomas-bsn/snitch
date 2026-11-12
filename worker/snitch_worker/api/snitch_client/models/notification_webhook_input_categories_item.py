from enum import StrEnum

class NotificationWebhookInputCategoriesItem(StrEnum):
    ACCESS_REQUESTS = "access_requests"
    HUMAN_ACTIONS = "human_actions"
    JOB_BLOCKED = "job_blocked"
    JOB_FAILURES = "job_failures"
    WORKER_OFFLINE = "worker_offline"

    def __str__(self) -> str:
        return str(self.value)
