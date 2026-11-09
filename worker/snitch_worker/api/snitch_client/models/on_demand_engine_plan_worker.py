from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.on_demand_engine_plan_worker_status import OnDemandEnginePlanWorkerStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="OnDemandEnginePlanWorker")



@_attrs_define
class OnDemandEnginePlanWorker:
    """
        Attributes:
            worker_id (str):
            status (OnDemandEnginePlanWorkerStatus):
            cause_code (str | Unset): Cause stable du refus (platform_incompatible, cuda_incompatible, insufficient_ram,
                insufficient_vram, insufficient_disk, engine_order_in_progress, engine_already_installed, worker_not_found).
            required_bytes (int | Unset):
            available_bytes (int | Unset):
     """

    worker_id: str
    status: OnDemandEnginePlanWorkerStatus
    cause_code: str | Unset = UNSET
    required_bytes: int | Unset = UNSET
    available_bytes: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        worker_id = self.worker_id

        status = self.status.value

        cause_code = self.cause_code

        required_bytes = self.required_bytes

        available_bytes = self.available_bytes


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "status": status,
        })
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code
        if required_bytes is not UNSET:
            field_dict["required_bytes"] = required_bytes
        if available_bytes is not UNSET:
            field_dict["available_bytes"] = available_bytes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        status = OnDemandEnginePlanWorkerStatus(d.pop("status"))




        cause_code = d.pop("cause_code", UNSET)

        required_bytes = d.pop("required_bytes", UNSET)

        available_bytes = d.pop("available_bytes", UNSET)

        on_demand_engine_plan_worker = cls(
            worker_id=worker_id,
            status=status,
            cause_code=cause_code,
            required_bytes=required_bytes,
            available_bytes=available_bytes,
        )


        on_demand_engine_plan_worker.additional_properties = d
        return on_demand_engine_plan_worker

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
