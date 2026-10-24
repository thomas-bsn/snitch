from enum import StrEnum

class SearchTranscriptsMode(StrEnum):
    EXACT = "exact"
    FULL_TEXT = "full_text"

    def __str__(self) -> str:
        return str(self.value)
