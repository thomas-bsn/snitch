from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerPulseInput")



@_attrs_define
class WorkerPulseInput:
    """
        Attributes:
            capacity (int):
     """

    capacity: int





    def to_dict(self) -> dict[str, Any]:
        capacity = self.capacity


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capacity": capacity,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capacity = d.pop("capacity")

        worker_pulse_input = cls(
            capacity=capacity,
        )

        return worker_pulse_input
