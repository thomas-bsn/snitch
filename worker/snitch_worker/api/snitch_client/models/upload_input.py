from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.upload_input_visibility import UploadInputVisibility
from ..types import UNSET, Unset






T = TypeVar("T", bound="UploadInput")



@_attrs_define
class UploadInput:
    """
        Attributes:
            filename (str):
            size_bytes (int):
            visibility (UploadInputVisibility | Unset):  Default: UploadInputVisibility.SHARED.
     """

    filename: str
    size_bytes: int
    visibility: UploadInputVisibility | Unset = UploadInputVisibility.SHARED





    def to_dict(self) -> dict[str, Any]:
        filename = self.filename

        size_bytes = self.size_bytes

        visibility: str | Unset = UNSET
        if not isinstance(self.visibility, Unset):
            visibility = self.visibility.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "filename": filename,
            "size_bytes": size_bytes,
        })
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        filename = d.pop("filename")

        size_bytes = d.pop("size_bytes")

        _visibility = d.pop("visibility", UNSET)
        visibility: UploadInputVisibility | Unset
        if isinstance(_visibility,  Unset):
            visibility = UNSET
        else:
            visibility = UploadInputVisibility(_visibility)




        upload_input = cls(
            filename=filename,
            size_bytes=size_bytes,
            visibility=visibility,
        )

        return upload_input
