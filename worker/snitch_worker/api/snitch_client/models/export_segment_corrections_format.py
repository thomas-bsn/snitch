from enum import StrEnum

class ExportSegmentCorrectionsFormat(StrEnum):
    JSON = "json"
    NDJSON = "ndjson"

    def __str__(self) -> str:
        return str(self.value)
