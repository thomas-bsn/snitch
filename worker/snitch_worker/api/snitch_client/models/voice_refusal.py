from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRefusal")



@_attrs_define
class VoiceRefusal:
    """
        Attributes:
            local_speaker_id (int):
            person_id (int):
            refused (bool):
     """

    local_speaker_id: int
    person_id: int
    refused: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        person_id = self.person_id

        refused = self.refused


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "person_id": person_id,
            "refused": refused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        person_id = d.pop("person_id")

        refused = d.pop("refused")

        voice_refusal = cls(
            local_speaker_id=local_speaker_id,
            person_id=person_id,
            refused=refused,
        )


        voice_refusal.additional_properties = d
        return voice_refusal

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
