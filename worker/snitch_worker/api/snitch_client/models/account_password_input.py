from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AccountPasswordInput")



@_attrs_define
class AccountPasswordInput:
    """
        Attributes:
            new_password (str):
            current_password (str | Unset):
     """

    new_password: str
    current_password: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        new_password = self.new_password

        current_password = self.current_password


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "new_password": new_password,
        })
        if current_password is not UNSET:
            field_dict["current_password"] = current_password

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        new_password = d.pop("new_password")

        current_password = d.pop("current_password", UNSET)

        account_password_input = cls(
            new_password=new_password,
            current_password=current_password,
        )

        return account_password_input
