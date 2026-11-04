from enum import StrEnum

class SpeakerCoverageCoverageStatus(StrEnum):
    NO_TRANSCRIPT = "no_transcript"
    OUTSIDE_TRANSCRIPT = "outside_transcript"
    SPEECH_WITHOUT_WORDS = "speech_without_words"
    WORDS = "words"

    def __str__(self) -> str:
        return str(self.value)
