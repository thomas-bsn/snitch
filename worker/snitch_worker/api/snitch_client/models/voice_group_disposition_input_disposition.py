from enum import StrEnum

class VoiceGroupDispositionInputDisposition(StrEnum):
    IGNORED = "ignored"
    RANDOM = "random"

    def __str__(self) -> str:
        return str(self.value)
