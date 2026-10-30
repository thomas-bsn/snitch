from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SetVideoRootBody")



@_attrs_define
class SetVideoRootBody:
    """
        Attributes:
            path (str):
     """

    path: str





    def to_dict(self) -> dict[str, Any]:
        path = self.path


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "path": path,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path")

        set_video_root_body = cls(
            path=path,
        )

        return set_video_root_body
