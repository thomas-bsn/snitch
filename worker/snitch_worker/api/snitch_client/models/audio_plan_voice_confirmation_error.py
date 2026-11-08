from enum import StrEnum

class AudioPlanVoiceConfirmationError(StrEnum):
    VOICE_REDIARIZATION_CONFIRMATION_REQUIRED = "voice_rediarization_confirmation_required"

    def __str__(self) -> str:
        return str(self.value)
