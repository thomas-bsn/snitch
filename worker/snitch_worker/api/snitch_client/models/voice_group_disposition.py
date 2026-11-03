from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_group_disposition_disposition import VoiceGroupDispositionDisposition






T = TypeVar("T", bound="VoiceGroupDisposition")



@_attrs_define
class VoiceGroupDisposition:
    """
        Attributes:
            group_id (int):
            disposition (VoiceGroupDispositionDisposition):
            voices (int):
     """

    group_id: int
    disposition: VoiceGroupDispositionDisposition
    voices: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        group_id = self.group_id

        disposition = self.disposition.value

        voices = self.voices


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "group_id": group_id,
            "disposition": disposition,
            "voices": voices,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        group_id = d.pop("group_id")

        disposition = VoiceGroupDispositionDisposition(d.pop("disposition"))




        voices = d.pop("voices")

        voice_group_disposition = cls(
            group_id=group_id,
            disposition=disposition,
            voices=voices,
        )


        voice_group_disposition.additional_properties = d
        return voice_group_disposition

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
