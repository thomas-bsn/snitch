from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.upload_status import UploadStatus
from ..models.upload_visibility import UploadVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="Upload")



@_attrs_define
class Upload:
    """
        Attributes:
            id (str):
            original_name (str):
            size_bytes (int):
            offset_bytes (int):
            chunk_size_bytes (int):
            visibility (UploadVisibility):
            status (UploadStatus):
            expires_at (datetime.datetime):
            clip_id (int | Unset):
            sha256 (str | Unset):
     """

    id: str
    original_name: str
    size_bytes: int
    offset_bytes: int
    chunk_size_bytes: int
    visibility: UploadVisibility
    status: UploadStatus
    expires_at: datetime.datetime
    clip_id: int | Unset = UNSET
    sha256: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        original_name = self.original_name

        size_bytes = self.size_bytes

        offset_bytes = self.offset_bytes

        chunk_size_bytes = self.chunk_size_bytes

        visibility = self.visibility.value

        status = self.status.value

        expires_at = self.expires_at.isoformat()

        clip_id = self.clip_id

        sha256 = self.sha256


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "original_name": original_name,
            "size_bytes": size_bytes,
            "offset_bytes": offset_bytes,
            "chunk_size_bytes": chunk_size_bytes,
            "visibility": visibility,
            "status": status,
            "expires_at": expires_at,
        })
        if clip_id is not UNSET:
            field_dict["clip_id"] = clip_id
        if sha256 is not UNSET:
            field_dict["sha256"] = sha256

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        original_name = d.pop("original_name")

        size_bytes = d.pop("size_bytes")

        offset_bytes = d.pop("offset_bytes")

        chunk_size_bytes = d.pop("chunk_size_bytes")

        visibility = UploadVisibility(d.pop("visibility"))




        status = UploadStatus(d.pop("status"))




        expires_at = datetime.datetime.fromisoformat(d.pop("expires_at"))




        clip_id = d.pop("clip_id", UNSET)

        sha256 = d.pop("sha256", UNSET)

        upload = cls(
            id=id,
            original_name=original_name,
            size_bytes=size_bytes,
            offset_bytes=offset_bytes,
            chunk_size_bytes=chunk_size_bytes,
            visibility=visibility,
            status=status,
            expires_at=expires_at,
            clip_id=clip_id,
            sha256=sha256,
        )


        upload.additional_properties = d
        return upload

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
