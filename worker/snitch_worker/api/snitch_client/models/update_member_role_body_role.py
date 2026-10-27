from enum import StrEnum

class UpdateMemberRoleBodyRole(StrEnum):
    ADMIN = "admin"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)
