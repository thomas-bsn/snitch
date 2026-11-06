from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AttributionInput")



@_attrs_define
class AttributionInput:
    """ Exactement un de person_id ou new_person_name.

        Attributes:
            person_id (int | Unset):
            new_person_name (str | Unset):
            allow_homonym (bool | Unset): Avec new_person_name : crée la Personne même si une Personne de même nom existe
                (409 person_exists sinon).
            expected_person_id (int | Unset): Correction uniquement : Personne actuellement validée telle que vue par
                l'utilisateur.
     """

    person_id: int | Unset = UNSET
    new_person_name: str | Unset = UNSET
    allow_homonym: bool | Unset = UNSET
    expected_person_id: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        person_id = self.person_id

        new_person_name = self.new_person_name

        allow_homonym = self.allow_homonym

        expected_person_id = self.expected_person_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if person_id is not UNSET:
            field_dict["person_id"] = person_id
        if new_person_name is not UNSET:
            field_dict["new_person_name"] = new_person_name
        if allow_homonym is not UNSET:
            field_dict["allow_homonym"] = allow_homonym
        if expected_person_id is not UNSET:
            field_dict["expected_person_id"] = expected_person_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        person_id = d.pop("person_id", UNSET)

        new_person_name = d.pop("new_person_name", UNSET)

        allow_homonym = d.pop("allow_homonym", UNSET)

        expected_person_id = d.pop("expected_person_id", UNSET)

        attribution_input = cls(
            person_id=person_id,
            new_person_name=new_person_name,
            allow_homonym=allow_homonym,
            expected_person_id=expected_person_id,
        )

        return attribution_input
