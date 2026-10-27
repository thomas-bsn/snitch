from enum import StrEnum

class MemberRole(StrEnum):
    ADMIN = "admin"
    SUPER_ADMIN = "super_admin"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)
