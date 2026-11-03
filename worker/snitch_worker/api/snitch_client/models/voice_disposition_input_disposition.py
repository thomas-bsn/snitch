from enum import StrEnum

class VoiceDispositionInputDisposition(StrEnum):
    IGNORED = "ignored"
    RANDOM = "random"

    def __str__(self) -> str:
        return str(self.value)
