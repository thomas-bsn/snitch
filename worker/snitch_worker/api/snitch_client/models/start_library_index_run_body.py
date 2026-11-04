from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.start_library_index_run_body_mode import StartLibraryIndexRunBodyMode






T = TypeVar("T", bound="StartLibraryIndexRunBody")



@_attrs_define
class StartLibraryIndexRunBody:
    """
        Attributes:
            mode (StartLibraryIndexRunBodyMode):
     """

    mode: StartLibraryIndexRunBodyMode





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = StartLibraryIndexRunBodyMode(d.pop("mode"))




        start_library_index_run_body = cls(
            mode=mode,
        )

        return start_library_index_run_body
