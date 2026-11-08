from enum import StrEnum

class AudioTrackDiagnosticStatus(StrEnum):
    AMBIGUOUS = "ambiguous"
    FAILED = "failed"
    NO_SPEECH = "no_speech"
    SPEECH = "speech"
    UNEXAMINED = "unexamined"

    def __str__(self) -> str:
        return str(self.value)
