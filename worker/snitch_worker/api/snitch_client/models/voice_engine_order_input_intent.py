from enum import StrEnum

class VoiceEngineOrderInputIntent(StrEnum):
    INSTALL = "install"
    UNINSTALL = "uninstall"

    def __str__(self) -> str:
        return str(self.value)
