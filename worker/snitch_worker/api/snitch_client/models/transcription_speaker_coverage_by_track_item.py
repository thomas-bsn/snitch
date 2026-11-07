from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.speaker_coverage import SpeakerCoverage





T = TypeVar("T", bound="TranscriptionSpeakerCoverageByTrackItem")



@_attrs_define
class TranscriptionSpeakerCoverageByTrackItem:
    """
        Attributes:
            audio_stream_index (int):
            coverage (list[SpeakerCoverage]):
     """

    audio_stream_index: int
    coverage: list[SpeakerCoverage]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.speaker_coverage import SpeakerCoverage # noqa: PLC0415
        audio_stream_index = self.audio_stream_index

        coverage = []
        for coverage_item_data in self.coverage:
            coverage_item = coverage_item_data.to_dict()
            coverage.append(coverage_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "audio_stream_index": audio_stream_index,
            "coverage": coverage,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.speaker_coverage import SpeakerCoverage # noqa: PLC0415
        d = dict(src_dict)
        audio_stream_index = d.pop("audio_stream_index")

        coverage = []
        _coverage = d.pop("coverage")
        for coverage_item_data in (_coverage):
            coverage_item = SpeakerCoverage.from_dict(coverage_item_data)



            coverage.append(coverage_item)


        transcription_speaker_coverage_by_track_item = cls(
            audio_stream_index=audio_stream_index,
            coverage=coverage,
        )


        transcription_speaker_coverage_by_track_item.additional_properties = d
        return transcription_speaker_coverage_by_track_item

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
