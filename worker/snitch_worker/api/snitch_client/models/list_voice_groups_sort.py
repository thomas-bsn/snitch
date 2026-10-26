from enum import StrEnum

class ListVoiceGroupsSort(StrEnum):
    CLIPS = "clips"
    MEMBERS = "members"

    def __str__(self) -> str:
        return str(self.value)
