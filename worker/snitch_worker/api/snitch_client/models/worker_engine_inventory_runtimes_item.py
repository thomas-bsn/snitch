from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerEngineInventoryRuntimesItem")



@_attrs_define
class WorkerEngineInventoryRuntimesItem:
    """
        Attributes:
            engine (str):
            local_identity (str):
     """

    engine: str
    local_identity: str





    def to_dict(self) -> dict[str, Any]:
        engine = self.engine

        local_identity = self.local_identity


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "engine": engine,
            "local_identity": local_identity,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        engine = d.pop("engine")

        local_identity = d.pop("local_identity")

        worker_engine_inventory_runtimes_item = cls(
            engine=engine,
            local_identity=local_identity,
        )

        return worker_engine_inventory_runtimes_item
