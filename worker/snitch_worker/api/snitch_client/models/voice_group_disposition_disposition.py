from enum import StrEnum

class VoiceGroupDispositionDisposition(StrEnum):
    IGNORED = "ignored"
    RANDOM = "random"

    def __str__(self) -> str:
        return str(self.value)
