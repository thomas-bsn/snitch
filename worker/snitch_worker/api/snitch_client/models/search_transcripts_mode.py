from enum import StrEnum

class SearchTranscriptsMode(StrEnum):
    EXACT = "exact"
    FULL_TEXT = "full_text"
    HYBRID = "hybrid"

    def __str__(self) -> str:
        return str(self.value)
