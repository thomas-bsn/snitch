from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LibraryFolder")



@_attrs_define
class LibraryFolder:
    """
        Attributes:
            source_id (int):
            source (str):
            name (str):
            path (str):
            private (bool): Dossier rendu privé par le Super-administrateur
            locked (bool): Privé et non ouvrable par l'utilisateur courant : seul son nom est exposé
     """

    source_id: int
    source: str
    name: str
    path: str
    private: bool
    locked: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        source_id = self.source_id

        source = self.source

        name = self.name

        path = self.path

        private = self.private

        locked = self.locked


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source_id": source_id,
            "source": source,
            "name": name,
            "path": path,
            "private": private,
            "locked": locked,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source_id = d.pop("source_id")

        source = d.pop("source")

        name = d.pop("name")

        path = d.pop("path")

        private = d.pop("private")

        locked = d.pop("locked")

        library_folder = cls(
            source_id=source_id,
            source=source,
            name=name,
            path=path,
            private=private,
            locked=locked,
        )


        library_folder.additional_properties = d
        return library_folder

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
