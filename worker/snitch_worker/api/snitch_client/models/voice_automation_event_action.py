from enum import StrEnum

class VoiceAutomationEventAction(StrEnum):
    DISABLE = "disable"
    ENABLE = "enable"

    def __str__(self) -> str:
        return str(self.value)
