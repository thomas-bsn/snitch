from enum import StrEnum

class PersonLinkRequestStatus(StrEnum):
    APPROVED = "approved"
    PENDING = "pending"
    REFUSED = "refused"

    def __str__(self) -> str:
        return str(self.value)
