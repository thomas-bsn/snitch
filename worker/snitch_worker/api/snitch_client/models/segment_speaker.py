from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SegmentSpeaker")



@_attrs_define
class SegmentSpeaker:
    """
        Attributes:
            transcription_job_id (int):
            index (int):
            version (int):
            local_speaker_id (int | None): Locuteur local effectif du segment (null sans locuteur déduit).
            local_speaker_label (None | str):
            corrected (bool): Vrai si le locuteur est désigné par un utilisateur, faux s'il est déduit.
     """

    transcription_job_id: int
    index: int
    version: int
    local_speaker_id: int | None
    local_speaker_label: None | str
    corrected: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        transcription_job_id = self.transcription_job_id

        index = self.index

        version = self.version

        local_speaker_id: int | None
        local_speaker_id = self.local_speaker_id

        local_speaker_label: None | str
        local_speaker_label = self.local_speaker_label

        corrected = self.corrected


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription_job_id": transcription_job_id,
            "index": index,
            "version": version,
            "local_speaker_id": local_speaker_id,
            "local_speaker_label": local_speaker_label,
            "corrected": corrected,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        transcription_job_id = d.pop("transcription_job_id")

        index = d.pop("index")

        version = d.pop("version")

        def _parse_local_speaker_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        local_speaker_id = _parse_local_speaker_id(d.pop("local_speaker_id"))


        def _parse_local_speaker_label(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        local_speaker_label = _parse_local_speaker_label(d.pop("local_speaker_label"))


        corrected = d.pop("corrected")

        segment_speaker = cls(
            transcription_job_id=transcription_job_id,
            index=index,
            version=version,
            local_speaker_id=local_speaker_id,
            local_speaker_label=local_speaker_label,
            corrected=corrected,
        )


        segment_speaker.additional_properties = d
        return segment_speaker

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
