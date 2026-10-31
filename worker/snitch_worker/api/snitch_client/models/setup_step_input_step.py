from enum import StrEnum

class SetupStepInputStep(StrEnum):
    DISCORD = "discord"
    IMPORT = "import"
    LOCATION = "location"
    SEMANTIC_SEARCH = "semantic_search"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
