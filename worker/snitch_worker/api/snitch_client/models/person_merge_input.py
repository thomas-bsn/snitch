from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PersonMergeInput")



@_attrs_define
class PersonMergeInput:
    """
        Attributes:
            target_person_id (int): Personne qui reste ; celle du chemin est absorbée.
     """

    target_person_id: int





    def to_dict(self) -> dict[str, Any]:
        target_person_id = self.target_person_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "target_person_id": target_person_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        target_person_id = d.pop("target_person_id")

        person_merge_input = cls(
            target_person_id=target_person_id,
        )

        return person_merge_input
