from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRefusalInput")



@_attrs_define
class VoiceRefusalInput:
    """
        Attributes:
            person_id (int):
     """

    person_id: int





    def to_dict(self) -> dict[str, Any]:
        person_id = self.person_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "person_id": person_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        person_id = d.pop("person_id")

        voice_refusal_input = cls(
            person_id=person_id,
        )

        return voice_refusal_input
