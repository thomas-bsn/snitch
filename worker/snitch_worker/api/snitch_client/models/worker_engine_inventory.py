from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_engine_inventory_runtimes_item import WorkerEngineInventoryRuntimesItem





T = TypeVar("T", bound="WorkerEngineInventory")



@_attrs_define
class WorkerEngineInventory:
    """
        Attributes:
            runtimes (list[WorkerEngineInventoryRuntimesItem]):
     """

    runtimes: list[WorkerEngineInventoryRuntimesItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_engine_inventory_runtimes_item import WorkerEngineInventoryRuntimesItem # noqa: PLC0415
        runtimes = []
        for runtimes_item_data in self.runtimes:
            runtimes_item = runtimes_item_data.to_dict()
            runtimes.append(runtimes_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "runtimes": runtimes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_engine_inventory_runtimes_item import WorkerEngineInventoryRuntimesItem # noqa: PLC0415
        d = dict(src_dict)
        runtimes = []
        _runtimes = d.pop("runtimes")
        for runtimes_item_data in (_runtimes):
            runtimes_item = WorkerEngineInventoryRuntimesItem.from_dict(runtimes_item_data)



            runtimes.append(runtimes_item)


        worker_engine_inventory = cls(
            runtimes=runtimes,
        )

        return worker_engine_inventory
