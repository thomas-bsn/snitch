from enum import StrEnum

class VoiceCalibrationModeWithheld(StrEnum):
    NO_SAFE_THRESHOLD = "no_safe_threshold"
    TOO_FEW = "too_few"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
