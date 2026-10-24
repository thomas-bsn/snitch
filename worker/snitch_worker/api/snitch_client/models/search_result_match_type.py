from enum import StrEnum

class SearchResultMatchType(StrEnum):
    APPROXIMATE = "approximate"
    EXACT = "exact"
    FULL_TEXT = "full_text"

    def __str__(self) -> str:
        return str(self.value)
