from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ScanVideoSelectionResponse200")



@_attrs_define
class ScanVideoSelectionResponse200:
    """
        Attributes:
            paths (list[str]):
            queued (bool):
     """

    paths: list[str]
    queued: bool





    def to_dict(self) -> dict[str, Any]:
        paths = self.paths



        queued = self.queued


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paths": paths,
            "queued": queued,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paths = cast(list[str], d.pop("paths"))


        queued = d.pop("queued")

        scan_video_selection_response_200 = cls(
            paths=paths,
            queued=queued,
        )

        return scan_video_selection_response_200
