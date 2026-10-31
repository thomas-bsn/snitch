from enum import StrEnum

class ClipOriginsItemKind(StrEnum):
    SOURCE = "source"
    UPLOAD = "upload"

    def __str__(self) -> str:
        return str(self.value)
