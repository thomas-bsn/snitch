from enum import StrEnum

class TranscriptionSettingsStateStatus(StrEnum):
    DISABLED = "disabled"
    UNCONFIGURED = "unconfigured"
    UNTESTED = "untested"

    def __str__(self) -> str:
        return str(self.value)
