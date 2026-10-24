from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.source_health import SourceHealth
from ..models.source_visibility import SourceVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="Source")



@_attrs_define
class Source:
    """
        Attributes:
            id (int):
            name (str):
            path (str): Visible uniquement aux administrateurs.
            visibility (SourceVisibility):
            enabled (bool):
            health (SourceHealth):
            last_scan_at (datetime.datetime | None | Unset):
     """

    id: int
    name: str
    path: str
    visibility: SourceVisibility
    enabled: bool
    health: SourceHealth
    last_scan_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        path = self.path

        visibility = self.visibility.value

        enabled = self.enabled

        health = self.health.value

        last_scan_at: None | str | Unset
        if isinstance(self.last_scan_at, Unset):
            last_scan_at = UNSET
        elif isinstance(self.last_scan_at, datetime.datetime):
            last_scan_at = self.last_scan_at.isoformat()
        else:
            last_scan_at = self.last_scan_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "path": path,
            "visibility": visibility,
            "enabled": enabled,
            "health": health,
        })
        if last_scan_at is not UNSET:
            field_dict["last_scan_at"] = last_scan_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        path = d.pop("path")

        visibility = SourceVisibility(d.pop("visibility"))




        enabled = d.pop("enabled")

        health = SourceHealth(d.pop("health"))




        def _parse_last_scan_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_scan_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_scan_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        last_scan_at = _parse_last_scan_at(d.pop("last_scan_at", UNSET))


        source = cls(
            id=id,
            name=name,
            path=path,
            visibility=visibility,
            enabled=enabled,
            health=health,
            last_scan_at=last_scan_at,
        )


        source.additional_properties = d
        return source

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
