from enum import StrEnum

class DiscordBotStatusStatus(StrEnum):
    CONNECTED = "connected"
    ERROR = "error"
    NOT_CONFIGURED = "not_configured"

    def __str__(self) -> str:
        return str(self.value)
