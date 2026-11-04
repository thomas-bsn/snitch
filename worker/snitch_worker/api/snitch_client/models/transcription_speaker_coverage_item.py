from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcription_speaker_coverage_item_coverage_status import TranscriptionSpeakerCoverageItemCoverageStatus






T = TypeVar("T", bound="TranscriptionSpeakerCoverageItem")



@_attrs_define
class TranscriptionSpeakerCoverageItem:
    """
        Attributes:
            local_speaker_id (int):
            coverage_status (TranscriptionSpeakerCoverageItemCoverageStatus): words : au moins un segment lui est attribué ;
                speech_without_words : il parle sous des segments attribués à d'autres ; no_transcript : aucun segment ne
                recouvre ses tours.
            speaking_seconds (float):
            overlapped_seconds (float): Parole sous des segments attribués à un autre locuteur.
            attributed_segments (int):
     """

    local_speaker_id: int
    coverage_status: TranscriptionSpeakerCoverageItemCoverageStatus
    speaking_seconds: float
    overlapped_seconds: float
    attributed_segments: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        coverage_status = self.coverage_status.value

        speaking_seconds = self.speaking_seconds

        overlapped_seconds = self.overlapped_seconds

        attributed_segments = self.attributed_segments


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "coverage_status": coverage_status,
            "speaking_seconds": speaking_seconds,
            "overlapped_seconds": overlapped_seconds,
            "attributed_segments": attributed_segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        coverage_status = TranscriptionSpeakerCoverageItemCoverageStatus(d.pop("coverage_status"))




        speaking_seconds = d.pop("speaking_seconds")

        overlapped_seconds = d.pop("overlapped_seconds")

        attributed_segments = d.pop("attributed_segments")

        transcription_speaker_coverage_item = cls(
            local_speaker_id=local_speaker_id,
            coverage_status=coverage_status,
            speaking_seconds=speaking_seconds,
            overlapped_seconds=overlapped_seconds,
            attributed_segments=attributed_segments,
        )


        transcription_speaker_coverage_item.additional_properties = d
        return transcription_speaker_coverage_item

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
