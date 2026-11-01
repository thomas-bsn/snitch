from enum import StrEnum

class WorkerRegistrationInstallCommandUnavailable(StrEnum):
    INSECURE_ORIGIN = "insecure_origin"
    INVALID_ORIGIN = "invalid_origin"

    def __str__(self) -> str:
        return str(self.value)
