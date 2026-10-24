from enum import StrEnum

class SearchResultMatchType(StrEnum):
    APPROXIMATE = "approximate"
    EXACT = "exact"
    FULL_TEXT = "full_text"
    HYBRID = "hybrid"
    SEMANTIC = "semantic"

    def __str__(self) -> str:
        return str(self.value)
