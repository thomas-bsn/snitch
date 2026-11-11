from enum import StrEnum

class VoiceMatchingInputSource(StrEnum):
    ADMINISTRATOR = "administrator"
    PROPOSAL = "proposal"

    def __str__(self) -> str:
        return str(self.value)
