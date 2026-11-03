from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.people_to_process_clip import PeopleToProcessClip
  from ..models.people_to_process_totals import PeopleToProcessTotals





T = TypeVar("T", bound="PeopleToProcess")



@_attrs_define
class PeopleToProcess:
    """
        Attributes:
            items (list[PeopleToProcessClip]):
            total (int):
            totals (PeopleToProcessTotals):
     """

    items: list[PeopleToProcessClip]
    total: int
    totals: PeopleToProcessTotals
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.people_to_process_clip import PeopleToProcessClip # noqa: PLC0415
        from ..models.people_to_process_totals import PeopleToProcessTotals # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        totals = self.totals.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "items": items,
            "total": total,
            "totals": totals,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.people_to_process_clip import PeopleToProcessClip # noqa: PLC0415
        from ..models.people_to_process_totals import PeopleToProcessTotals # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = PeopleToProcessClip.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        totals = PeopleToProcessTotals.from_dict(d.pop("totals"))




        people_to_process = cls(
            items=items,
            total=total,
            totals=totals,
        )


        people_to_process.additional_properties = d
        return people_to_process

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
