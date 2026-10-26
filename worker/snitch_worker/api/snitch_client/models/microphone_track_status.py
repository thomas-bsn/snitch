from enum import StrEnum

class MicrophoneTrackStatus(StrEnum):
    AMBIGUOUS_TRACK = "ambiguous_track"
    BLOCKED_BY_VALIDATION = "blocked_by_validation"
    CONFLICTING_SOURCES = "conflicting_sources"
    DESIGNATED = "designated"
    DISABLED = "disabled"
    NONE = "none"
    OWNER_PROFILE_DELETED = "owner_profile_deleted"
    TRACK_NOT_FOUND = "track_not_found"

    def __str__(self) -> str:
        return str(self.value)
