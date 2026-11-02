from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.analysis_queue_run import AnalysisQueueRun





T = TypeVar("T", bound="AnalysisRunHistory")



@_attrs_define
class AnalysisRunHistory:
    """
        Attributes:
            items (list[AnalysisQueueRun]):
            total (int):
            page (int):
            page_size (int):
     """

    items: list[AnalysisQueueRun]
    total: int
    page: int
    page_size: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        page = self.page

        page_size = self.page_size


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "items": items,
            "total": total,
            "page": page,
            "page_size": page_size,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AnalysisQueueRun.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        page = d.pop("page")

        page_size = d.pop("page_size")

        analysis_run_history = cls(
            items=items,
            total=total,
            page=page,
            page_size=page_size,
        )


        analysis_run_history.additional_properties = d
        return analysis_run_history

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
