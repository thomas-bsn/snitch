from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PersonSeparationInput")



@_attrs_define
class PersonSeparationInput:
    """
        Attributes:
            local_speaker_ids (list[int]):
     """

    local_speaker_ids: list[int]





    def to_dict(self) -> dict[str, Any]:
        local_speaker_ids = self.local_speaker_ids




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "local_speaker_ids": local_speaker_ids,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_ids = cast(list[int], d.pop("local_speaker_ids"))


        person_separation_input = cls(
            local_speaker_ids=local_speaker_ids,
        )

        return person_separation_input
