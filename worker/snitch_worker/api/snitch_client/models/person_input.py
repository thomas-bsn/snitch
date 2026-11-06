from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PersonInput")



@_attrs_define
class PersonInput:
    """
        Attributes:
            display_name (str):
            allow_homonym (bool | Unset): Vrai pour créer une Personne de même nom qu'une existante, après que l'utilisateur
                a refusé de la réutiliser.
     """

    display_name: str
    allow_homonym: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        display_name = self.display_name

        allow_homonym = self.allow_homonym


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "display_name": display_name,
        })
        if allow_homonym is not UNSET:
            field_dict["allow_homonym"] = allow_homonym

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        display_name = d.pop("display_name")

        allow_homonym = d.pop("allow_homonym", UNSET)

        person_input = cls(
            display_name=display_name,
            allow_homonym=allow_homonym,
        )

        return person_input
