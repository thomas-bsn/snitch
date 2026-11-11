from enum import StrEnum

class VoiceGroupThresholdsSource(StrEnum):
    ADMINISTRATOR = "administrator"
    MIGRATION = "migration"
    PROPOSAL = "proposal"

    def __str__(self) -> str:
        return str(self.value)
