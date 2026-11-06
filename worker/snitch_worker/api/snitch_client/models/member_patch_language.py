from enum import StrEnum

class MemberPatchLanguage(StrEnum):
    EN = "en"
    FR = "fr"

    def __str__(self) -> str:
        return str(self.value)
