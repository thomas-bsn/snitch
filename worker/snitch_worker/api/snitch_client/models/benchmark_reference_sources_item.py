from enum import StrEnum

class BenchmarkReferenceSourcesItem(StrEnum):
    IMPORT = "import"
    PRODUCTION = "production"

    def __str__(self) -> str:
        return str(self.value)
