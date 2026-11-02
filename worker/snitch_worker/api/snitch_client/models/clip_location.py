from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ClipLocation")



@_attrs_define
class ClipLocation:
    """
        Attributes:
            source (str): Nom de la source indexée.
            path (str | Unset): Chemin complet (racine de la source + chemin relatif). Absent pour un dossier privé.
            private (bool | Unset): Le fichier est dans un dossier privé : son chemin n'est pas envoyé.
     """

    source: str
    path: str | Unset = UNSET
    private: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        source = self.source

        path = self.path

        private = self.private


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source": source,
        })
        if path is not UNSET:
            field_dict["path"] = path
        if private is not UNSET:
            field_dict["private"] = private

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = d.pop("source")

        path = d.pop("path", UNSET)

        private = d.pop("private", UNSET)

        clip_location = cls(
            source=source,
            path=path,
            private=private,
        )


        clip_location.additional_properties = d
        return clip_location

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
