from enum import StrEnum

class SettingsUnreadableSecretsItem(StrEnum):
    DISCORD_CLIENT_ID = "discord_client_id"
    DISCORD_CLIENT_SECRET = "discord_client_secret"

    def __str__(self) -> str:
        return str(self.value)
