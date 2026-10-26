from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.master_key_export_origin import MasterKeyExportOrigin
from ..models.master_key_export_warning import MasterKeyExportWarning






T = TypeVar("T", bound="MasterKeyExport")



@_attrs_define
class MasterKeyExport:
    """
        Attributes:
            master_key (str): Clé de 32 octets encodée en base64, utilisable telle quelle comme valeur de MASTER_KEY.
            origin (MasterKeyExportOrigin):
            warning (MasterKeyExportWarning): Code de l'avertissement : cette clé ne se rejoue pas, la perdre rend les
                secrets chiffrés irrécupérables.
     """

    master_key: str
    origin: MasterKeyExportOrigin
    warning: MasterKeyExportWarning
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        master_key = self.master_key

        origin = self.origin.value

        warning = self.warning.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "master_key": master_key,
            "origin": origin,
            "warning": warning,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        master_key = d.pop("master_key")

        origin = MasterKeyExportOrigin(d.pop("origin"))




        warning = MasterKeyExportWarning(d.pop("warning"))




        master_key_export = cls(
            master_key=master_key,
            origin=origin,
            warning=warning,
        )


        master_key_export.additional_properties = d
        return master_key_export

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
