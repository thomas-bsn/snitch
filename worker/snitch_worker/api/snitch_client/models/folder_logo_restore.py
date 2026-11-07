from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="FolderLogoRestore")



@_attrs_define
class FolderLogoRestore:
    """
        Attributes:
            path (str):
            version (int):
     """

    path: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        path = self.path

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "path": path,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path")

        version = d.pop("version")

        folder_logo_restore = cls(
            path=path,
            version=version,
        )

        return folder_logo_restore
