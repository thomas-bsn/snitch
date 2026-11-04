from enum import StrEnum

class TranscriptionSpeakerCoverageItemCoverageStatus(StrEnum):
    NO_TRANSCRIPT = "no_transcript"
    SPEECH_WITHOUT_WORDS = "speech_without_words"
    WORDS = "words"

    def __str__(self) -> str:
        return str(self.value)
