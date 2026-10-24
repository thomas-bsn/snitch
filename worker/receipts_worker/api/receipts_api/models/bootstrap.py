from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="Bootstrap")



@_attrs_define
class Bootstrap:
    """
        Attributes:
            username (str):
            password (str):
            code (str):
     """

    username: str
    password: str
    code: str





    def to_dict(self) -> dict[str, Any]:
        username = self.username

        password = self.password

        code = self.code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "username": username,
            "password": password,
            "code": code,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        username = d.pop("username")

        password = d.pop("password")

        code = d.pop("code")

        bootstrap = cls(
            username=username,
            password=password,
            code=code,
        )

        return bootstrap
