from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="IndexRunPreview")



@_attrs_define
class IndexRunPreview:
    """
        Attributes:
            detected (int): Vidéos que l'utilisateur peut voir dans la Racine.
            already_indexed (int): Déjà dotées d'une fiche clip : jamais retraitées.
            new (int): Sans fiche clip : celles qu'une indexation traiterait.
     """

    detected: int
    already_indexed: int
    new: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        detected = self.detected

        already_indexed = self.already_indexed

        new = self.new


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "detected": detected,
            "already_indexed": already_indexed,
            "new": new,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detected = d.pop("detected")

        already_indexed = d.pop("already_indexed")

        new = d.pop("new")

        index_run_preview = cls(
            detected=detected,
            already_indexed=already_indexed,
            new=new,
        )


        index_run_preview.additional_properties = d
        return index_run_preview

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
