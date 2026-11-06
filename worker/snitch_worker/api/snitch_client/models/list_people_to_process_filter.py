from enum import StrEnum

class ListPeopleToProcessFilter(StrEnum):
    ALL = "all"
    GROUP = "group"
    TO_CONFIRM = "to_confirm"
    UNKNOWN_AMBIGUOUS = "unknown_ambiguous"

    def __str__(self) -> str:
        return str(self.value)
