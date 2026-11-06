from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.explorer_entry import ExplorerEntry





T = TypeVar("T", bound="ExplorerListing")



@_attrs_define
class ExplorerListing:
    """
        Attributes:
            source_id (int):
            root_configured (bool):
            items (list[ExplorerEntry]):
            total (int): Taille de la sélection entière, hors pagination
            states (list[str]):
     """

    source_id: int
    root_configured: bool
    items: list[ExplorerEntry]
    total: int
    states: list[str]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.explorer_entry import ExplorerEntry # noqa: PLC0415
        source_id = self.source_id

        root_configured = self.root_configured

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        states = self.states




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source_id": source_id,
            "root_configured": root_configured,
            "items": items,
            "total": total,
            "states": states,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.explorer_entry import ExplorerEntry # noqa: PLC0415
        d = dict(src_dict)
        source_id = d.pop("source_id")

        root_configured = d.pop("root_configured")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ExplorerEntry.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        states = cast(list[str], d.pop("states"))


        explorer_listing = cls(
            source_id=source_id,
            root_configured=root_configured,
            items=items,
            total=total,
            states=states,
        )


        explorer_listing.additional_properties = d
        return explorer_listing

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
