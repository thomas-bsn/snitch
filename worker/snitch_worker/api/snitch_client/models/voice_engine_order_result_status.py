from enum import StrEnum

class VoiceEngineOrderResultStatus(StrEnum):
    CREATED = "created"
    REFUSED = "refused"
    REPLAYED = "replayed"

    def __str__(self) -> str:
        return str(self.value)
