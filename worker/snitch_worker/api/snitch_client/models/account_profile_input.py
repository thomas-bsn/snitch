from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AccountProfileInput")



@_attrs_define
class AccountProfileInput:
    """
        Attributes:
            display_name (str):
            username (str):
     """

    display_name: str
    username: str





    def to_dict(self) -> dict[str, Any]:
        display_name = self.display_name

        username = self.username


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "display_name": display_name,
            "username": username,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        display_name = d.pop("display_name")

        username = d.pop("username")

        account_profile_input = cls(
            display_name=display_name,
            username=username,
        )

        return account_profile_input
