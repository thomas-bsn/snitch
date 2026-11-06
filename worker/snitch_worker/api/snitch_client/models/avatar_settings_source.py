from enum import StrEnum

class AvatarSettingsSource(StrEnum):
    CUSTOM = "custom"
    DISCORD = "discord"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
