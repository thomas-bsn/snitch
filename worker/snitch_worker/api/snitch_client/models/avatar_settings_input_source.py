from enum import StrEnum

class AvatarSettingsInputSource(StrEnum):
    CUSTOM = "custom"
    DISCORD = "discord"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
