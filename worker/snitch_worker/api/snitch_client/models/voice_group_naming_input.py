from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="VoiceGroupNamingInput")



@_attrs_define
class VoiceGroupNamingInput:
    """
        Attributes:
            local_speaker_ids (list[int]): Exactement les voix visibles du groupe que l'utilisateur a écoutées.
            person_id (int | Unset):
            new_person_name (str | Unset):
     """

    local_speaker_ids: list[int]
    person_id: int | Unset = UNSET
    new_person_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        local_speaker_ids = self.local_speaker_ids



        person_id = self.person_id

        new_person_name = self.new_person_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "local_speaker_ids": local_speaker_ids,
        })
        if person_id is not UNSET:
            field_dict["person_id"] = person_id
        if new_person_name is not UNSET:
            field_dict["new_person_name"] = new_person_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_ids = cast(list[int], d.pop("local_speaker_ids"))


        person_id = d.pop("person_id", UNSET)

        new_person_name = d.pop("new_person_name", UNSET)

        voice_group_naming_input = cls(
            local_speaker_ids=local_speaker_ids,
            person_id=person_id,
            new_person_name=new_person_name,
        )

        return voice_group_naming_input
