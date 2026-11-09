from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="OnDemandEnginePlanInput")



@_attrs_define
class OnDemandEnginePlanInput:
    """
        Attributes:
            family (str):
            worker_ids (list[str]):
            version (str | Unset): La version par défaut de la famille si omise.
     """

    family: str
    worker_ids: list[str]
    version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        family = self.family

        worker_ids = self.worker_ids



        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "family": family,
            "worker_ids": worker_ids,
        })
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        family = d.pop("family")

        worker_ids = cast(list[str], d.pop("worker_ids"))


        version = d.pop("version", UNSET)

        on_demand_engine_plan_input = cls(
            family=family,
            worker_ids=worker_ids,
            version=version,
        )

        return on_demand_engine_plan_input
