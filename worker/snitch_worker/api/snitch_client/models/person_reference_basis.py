from enum import StrEnum

class PersonReferenceBasis(StrEnum):
    BELOW_MEDIAN = "below_median"
    FEW_REFERENCES = "few_references"
    MEDIAN_OR_MORE = "median_or_more"
    NO_SAMPLE = "no_sample"
    SINGLE_CLIP = "single_clip"

    def __str__(self) -> str:
        return str(self.value)
