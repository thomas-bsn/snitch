from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PersonLinkRequestInput")



@_attrs_define
class PersonLinkRequestInput:
    """
        Attributes:
            person_id (int):
            message (str | Unset): Message facultatif à l'attention de l'Administrateur.
     """

    person_id: int
    message: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        person_id = self.person_id

        message = self.message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "person_id": person_id,
        })
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        person_id = d.pop("person_id")

        message = d.pop("message", UNSET)

        person_link_request_input = cls(
            person_id=person_id,
            message=message,
        )

        return person_link_request_input
