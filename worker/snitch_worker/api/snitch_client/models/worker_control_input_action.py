from enum import StrEnum

class WorkerControlInputAction(StrEnum):
    PAUSE = "pause"
    RESUME = "resume"

    def __str__(self) -> str:
        return str(self.value)
