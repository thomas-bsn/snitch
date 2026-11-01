from enum import StrEnum

class TranscriptionSettingsStateMode(StrEnum):
    DISABLED = "disabled"
    ENABLED = "enabled"
    UNCONFIGURED = "unconfigured"

    def __str__(self) -> str:
        return str(self.value)
