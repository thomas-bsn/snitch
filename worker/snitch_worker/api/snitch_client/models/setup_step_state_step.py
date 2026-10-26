from enum import StrEnum

class SetupStepStateStep(StrEnum):
    DISCORD = "discord"
    IMPORT = "import"
    SEMANTIC_SEARCH = "semantic_search"
    WORKER = "worker"

    def __str__(self) -> str:
        return str(self.value)
