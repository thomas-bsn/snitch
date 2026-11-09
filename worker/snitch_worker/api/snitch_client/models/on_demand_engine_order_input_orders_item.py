from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="OnDemandEngineOrderInputOrdersItem")



@_attrs_define
class OnDemandEngineOrderInputOrdersItem:
    """
        Attributes:
            worker_id (str):
            idempotency_key (str):
     """

    worker_id: str
    idempotency_key: str





    def to_dict(self) -> dict[str, Any]:
        worker_id = self.worker_id

        idempotency_key = self.idempotency_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "worker_id": worker_id,
            "idempotency_key": idempotency_key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        idempotency_key = d.pop("idempotency_key")

        on_demand_engine_order_input_orders_item = cls(
            worker_id=worker_id,
            idempotency_key=idempotency_key,
        )

        return on_demand_engine_order_input_orders_item
