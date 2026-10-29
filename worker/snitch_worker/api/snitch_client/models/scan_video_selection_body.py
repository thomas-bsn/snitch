from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ScanVideoSelectionBody")



@_attrs_define
class ScanVideoSelectionBody:
    """
        Attributes:
            paths (list[str]):
            confirm (bool):
     """

    paths: list[str]
    confirm: bool





    def to_dict(self) -> dict[str, Any]:
        paths = self.paths



        confirm = self.confirm


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paths": paths,
            "confirm": confirm,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paths = cast(list[str], d.pop("paths"))


        confirm = d.pop("confirm")

        scan_video_selection_body = cls(
            paths=paths,
            confirm=confirm,
        )

        return scan_video_selection_body
