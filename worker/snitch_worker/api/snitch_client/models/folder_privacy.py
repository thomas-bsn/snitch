from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="FolderPrivacy")



@_attrs_define
class FolderPrivacy:
    """
        Attributes:
            source_id (int):
            folder (str): Nom d'un dossier de premier niveau de la source
            private (bool):
     """

    source_id: int
    folder: str
    private: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        source_id = self.source_id

        folder = self.folder

        private = self.private


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source_id": source_id,
            "folder": folder,
            "private": private,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source_id = d.pop("source_id")

        folder = d.pop("folder")

        private = d.pop("private")

        folder_privacy = cls(
            source_id=source_id,
            folder=folder,
            private=private,
        )


        folder_privacy.additional_properties = d
        return folder_privacy

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
