from enum import StrEnum

class TaskCardColumn(StrEnum):
    COMPLETED = "completed"
    RECOGNITION = "recognition"
    SEARCH = "search"
    TRANSCRIPTION = "transcription"
    VOICES = "voices"

    def __str__(self) -> str:
        return str(self.value)
