from enum import StrEnum

class VoiceModelOrderHistoryItemAction(StrEnum):
    INSTALL = "install"
    REMOVE = "remove"

    def __str__(self) -> str:
        return str(self.value)
