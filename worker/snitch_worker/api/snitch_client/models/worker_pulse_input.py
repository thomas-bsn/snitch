from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WorkerPulseInput")



@_attrs_define
class WorkerPulseInput:
    """
        Attributes:
            capacity (int):
            supported_steps (list[str]):
     """

    capacity: int
    supported_steps: list[str]





    def to_dict(self) -> dict[str, Any]:
        capacity = self.capacity

        supported_steps = self.supported_steps




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capacity": capacity,
            "supported_steps": supported_steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capacity = d.pop("capacity")

        supported_steps = cast(list[str], d.pop("supported_steps"))


        worker_pulse_input = cls(
            capacity=capacity,
            supported_steps=supported_steps,
        )

        return worker_pulse_input
