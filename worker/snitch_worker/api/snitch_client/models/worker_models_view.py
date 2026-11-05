from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_models_row import WorkerModelsRow





T = TypeVar("T", bound="WorkerModelsView")



@_attrs_define
class WorkerModelsView:
    """
        Attributes:
            workers (list[WorkerModelsRow]):
     """

    workers: list[WorkerModelsRow]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_models_row import WorkerModelsRow # noqa: PLC0415
        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "workers": workers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_models_row import WorkerModelsRow # noqa: PLC0415
        d = dict(src_dict)
        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = WorkerModelsRow.from_dict(workers_item_data)



            workers.append(workers_item)


        worker_models_view = cls(
            workers=workers,
        )


        worker_models_view.additional_properties = d
        return worker_models_view

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
