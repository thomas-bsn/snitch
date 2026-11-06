from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_disposition_view_disposition import VoiceDispositionViewDisposition






T = TypeVar("T", bound="VoiceDispositionView")



@_attrs_define
class VoiceDispositionView:
    """
        Attributes:
            local_speaker_id (int):
            disposition (VoiceDispositionViewDisposition):
     """

    local_speaker_id: int
    disposition: VoiceDispositionViewDisposition
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        disposition = self.disposition.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "disposition": disposition,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        disposition = VoiceDispositionViewDisposition(d.pop("disposition"))




        voice_disposition_view = cls(
            local_speaker_id=local_speaker_id,
            disposition=disposition,
        )


        voice_disposition_view.additional_properties = d
        return voice_disposition_view

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
