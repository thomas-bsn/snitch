from enum import StrEnum

class VoiceDispositionViewDisposition(StrEnum):
    IGNORED = "ignored"
    RANDOM = "random"

    def __str__(self) -> str:
        return str(self.value)
