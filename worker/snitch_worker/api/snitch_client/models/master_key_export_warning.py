from enum import StrEnum

class MasterKeyExportWarning(StrEnum):
    MASTER_KEY_NOT_REPLAYABLE = "master_key_not_replayable"

    def __str__(self) -> str:
        return str(self.value)
