from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.on_demand_engine_plan_worker import OnDemandEnginePlanWorker





T = TypeVar("T", bound="OnDemandEnginePlan")



@_attrs_define
class OnDemandEnginePlan:
    """
        Attributes:
            family (str):
            version (str):
            accepted (int):
            disk_bytes (int): Disque cumulé des Workers acceptés.
            estimated_seconds (int): Durée estimée d'une installation ; les Workers installent en parallèle.
            workers (list[OnDemandEnginePlanWorker]):
     """

    family: str
    version: str
    accepted: int
    disk_bytes: int
    estimated_seconds: int
    workers: list[OnDemandEnginePlanWorker]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.on_demand_engine_plan_worker import OnDemandEnginePlanWorker # noqa: PLC0415
        family = self.family

        version = self.version

        accepted = self.accepted

        disk_bytes = self.disk_bytes

        estimated_seconds = self.estimated_seconds

        workers = []
        for workers_item_data in self.workers:
            workers_item = workers_item_data.to_dict()
            workers.append(workers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "family": family,
            "version": version,
            "accepted": accepted,
            "disk_bytes": disk_bytes,
            "estimated_seconds": estimated_seconds,
            "workers": workers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.on_demand_engine_plan_worker import OnDemandEnginePlanWorker # noqa: PLC0415
        d = dict(src_dict)
        family = d.pop("family")

        version = d.pop("version")

        accepted = d.pop("accepted")

        disk_bytes = d.pop("disk_bytes")

        estimated_seconds = d.pop("estimated_seconds")

        workers = []
        _workers = d.pop("workers")
        for workers_item_data in (_workers):
            workers_item = OnDemandEnginePlanWorker.from_dict(workers_item_data)



            workers.append(workers_item)


        on_demand_engine_plan = cls(
            family=family,
            version=version,
            accepted=accepted,
            disk_bytes=disk_bytes,
            estimated_seconds=estimated_seconds,
            workers=workers,
        )


        on_demand_engine_plan.additional_properties = d
        return on_demand_engine_plan

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
