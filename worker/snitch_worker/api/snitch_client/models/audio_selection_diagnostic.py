from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_selection_diagnostic_freshness import AudioSelectionDiagnosticFreshness
from ..models.audio_selection_diagnostic_status import AudioSelectionDiagnosticStatus






T = TypeVar("T", bound="AudioSelectionDiagnostic")



@_attrs_define
class AudioSelectionDiagnostic:
    """
        Attributes:
            track_index (int):
            status (AudioSelectionDiagnosticStatus):
            freshness (AudioSelectionDiagnosticFreshness):
            cause (str):
     """

    track_index: int
    status: AudioSelectionDiagnosticStatus
    freshness: AudioSelectionDiagnosticFreshness
    cause: str





    def to_dict(self) -> dict[str, Any]:
        track_index = self.track_index

        status = self.status.value

        freshness = self.freshness.value

        cause = self.cause


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "track_index": track_index,
            "status": status,
            "freshness": freshness,
            "cause": cause,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        track_index = d.pop("track_index")

        status = AudioSelectionDiagnosticStatus(d.pop("status"))




        freshness = AudioSelectionDiagnosticFreshness(d.pop("freshness"))




        cause = d.pop("cause")

        audio_selection_diagnostic = cls(
            track_index=track_index,
            status=status,
            freshness=freshness,
            cause=cause,
        )

        return audio_selection_diagnostic
