from enum import StrEnum

class SettingsMasterKeyOrigin(StrEnum):
    GENERATED = "generated"
    PROVIDED = "provided"

    def __str__(self) -> str:
        return str(self.value)
