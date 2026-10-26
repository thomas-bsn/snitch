from enum import StrEnum

class AccountPreferencesLanguage(StrEnum):
    EN = "en"
    FR = "fr"

    def __str__(self) -> str:
        return str(self.value)
