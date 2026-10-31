from enum import StrEnum

class ListPersonLinkRequestsStatus(StrEnum):
    APPROVED = "approved"
    PENDING = "pending"
    REFUSED = "refused"

    def __str__(self) -> str:
        return str(self.value)
