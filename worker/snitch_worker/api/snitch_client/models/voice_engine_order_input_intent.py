from enum import StrEnum

class VoiceEngineOrderInputIntent(StrEnum):
    INSTALL = "install"

    def __str__(self) -> str:
        return str(self.value)
