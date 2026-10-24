from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.source import Source





T = TypeVar("T", bound="SourceList")



@_attrs_define
class SourceList:
    """
        Attributes:
            allowed_roots (list[str]):
            items (list[Source]):
     """

    allowed_roots: list[str]
    items: list[Source]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.source import Source # noqa: PLC0415
        allowed_roots = self.allowed_roots



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "allowed_roots": allowed_roots,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.source import Source # noqa: PLC0415
        d = dict(src_dict)
        allowed_roots = cast(list[str], d.pop("allowed_roots"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = Source.from_dict(items_item_data)



            items.append(items_item)


        source_list = cls(
            allowed_roots=allowed_roots,
            items=items,
        )


        source_list.additional_properties = d
        return source_list

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
