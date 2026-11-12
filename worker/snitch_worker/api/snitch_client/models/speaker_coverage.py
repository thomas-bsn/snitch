from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.speaker_coverage_coverage_status import SpeakerCoverageCoverageStatus






T = TypeVar("T", bound="SpeakerCoverage")



@_attrs_define
class SpeakerCoverage:
    """
        Attributes:
            local_speaker_id (int):
            coverage_status (SpeakerCoverageCoverageStatus): words : il tient une part d'au moins un segment ;
                speech_without_words : il parle sous des segments dont il ne tient aucune part ; no_transcript : aucun segment
                ne recouvre ses tours, dans un trou de la transcription ; outside_transcript : aucun segment ne recouvre ses
                tours, avant ou après toute la transcription (transcription tronquée, décalage, mauvaise piste).
            speaking_seconds (float): Parole détectée, tours qui se chevauchent comptés une fois.
            overlapped_seconds (float): Parole sous des segments dont il ne tient aucune part.
            uncovered_seconds (float): Parole sous aucun segment.
            attributed_segments (int): Segments dont il tient au moins une part.
     """

    local_speaker_id: int
    coverage_status: SpeakerCoverageCoverageStatus
    speaking_seconds: float
    overlapped_seconds: float
    uncovered_seconds: float
    attributed_segments: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        coverage_status = self.coverage_status.value

        speaking_seconds = self.speaking_seconds

        overlapped_seconds = self.overlapped_seconds

        uncovered_seconds = self.uncovered_seconds

        attributed_segments = self.attributed_segments


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "coverage_status": coverage_status,
            "speaking_seconds": speaking_seconds,
            "overlapped_seconds": overlapped_seconds,
            "uncovered_seconds": uncovered_seconds,
            "attributed_segments": attributed_segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        coverage_status = SpeakerCoverageCoverageStatus(d.pop("coverage_status"))




        speaking_seconds = d.pop("speaking_seconds")

        overlapped_seconds = d.pop("overlapped_seconds")

        uncovered_seconds = d.pop("uncovered_seconds")

        attributed_segments = d.pop("attributed_segments")

        speaker_coverage = cls(
            local_speaker_id=local_speaker_id,
            coverage_status=coverage_status,
            speaking_seconds=speaking_seconds,
            overlapped_seconds=overlapped_seconds,
            uncovered_seconds=uncovered_seconds,
            attributed_segments=attributed_segments,
        )


        speaker_coverage.additional_properties = d
        return speaker_coverage

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
