from enum import StrEnum

class TranscriptionSettingsInputMode(StrEnum):
    DISABLED = "disabled"
    ENABLED = "enabled"

    def __str__(self) -> str:
        return str(self.value)
