from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.upload_publication import UploadPublication
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
            destination (str | Unset): Sous-dossier relatif demandé à la création, s'il y en a un.
            publication (UploadPublication | Unset): État de la publication vers la racine vidéo. Absent sans destination.
            publication_error (str | Unset): Code stable expliquant la dernière publication échouée.
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
    destination: str | Unset = UNSET
    publication: UploadPublication | Unset = UNSET
    publication_error: str | Unset = UNSET
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

        destination = self.destination

        publication: str | Unset = UNSET
        if not isinstance(self.publication, Unset):
            publication = self.publication.value


        publication_error = self.publication_error


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
        if destination is not UNSET:
            field_dict["destination"] = destination
        if publication is not UNSET:
            field_dict["publication"] = publication
        if publication_error is not UNSET:
            field_dict["publication_error"] = publication_error

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

        destination = d.pop("destination", UNSET)

        _publication = d.pop("publication", UNSET)
        publication: UploadPublication | Unset
        if isinstance(_publication,  Unset):
            publication = UNSET
        else:
            publication = UploadPublication(_publication)




        publication_error = d.pop("publication_error", UNSET)

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
            destination=destination,
            publication=publication,
            publication_error=publication_error,
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
