from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrowseVideoRootResponse200ItemsItem")



@_attrs_define
class BrowseVideoRootResponse200ItemsItem:
    """
        Attributes:
            path (str):
            directory (bool):
            size_bytes (int):
     """

    path: str
    directory: bool
    size_bytes: int





    def to_dict(self) -> dict[str, Any]:
        path = self.path

        directory = self.directory

        size_bytes = self.size_bytes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "path": path,
            "directory": directory,
            "size_bytes": size_bytes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path")

        directory = d.pop("directory")

        size_bytes = d.pop("size_bytes")

        browse_video_root_response_200_items_item = cls(
            path=path,
            directory=directory,
            size_bytes=size_bytes,
        )

        return browse_video_root_response_200_items_item
