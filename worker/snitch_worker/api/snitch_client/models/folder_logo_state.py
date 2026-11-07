from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.folder_logo_version import FolderLogoVersion





T = TypeVar("T", bound="FolderLogoState")



@_attrs_define
class FolderLogoState:
    """
        Attributes:
            path (str):
            custom (bool): Faux : le logo automatique est affiché, les versions restent dans l'historique
            versions (list[FolderLogoVersion]): La plus récente d'abord, quatre au maximum
            active_version (int | Unset): Version actuellement sélectionnée lorsque custom vaut true
     """

    path: str
    custom: bool
    versions: list[FolderLogoVersion]
    active_version: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.folder_logo_version import FolderLogoVersion # noqa: PLC0415
        path = self.path

        custom = self.custom

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)



        active_version = self.active_version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "path": path,
            "custom": custom,
            "versions": versions,
        })
        if active_version is not UNSET:
            field_dict["active_version"] = active_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.folder_logo_version import FolderLogoVersion # noqa: PLC0415
        d = dict(src_dict)
        path = d.pop("path")

        custom = d.pop("custom")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = FolderLogoVersion.from_dict(versions_item_data)



            versions.append(versions_item)


        active_version = d.pop("active_version", UNSET)

        folder_logo_state = cls(
            path=path,
            custom=custom,
            versions=versions,
            active_version=active_version,
        )


        folder_logo_state.additional_properties = d
        return folder_logo_state

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
