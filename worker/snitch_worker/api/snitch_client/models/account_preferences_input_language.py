from enum import StrEnum

class AccountPreferencesInputLanguage(StrEnum):
    EN = "en"
    FR = "fr"

    def __str__(self) -> str:
        return str(self.value)
