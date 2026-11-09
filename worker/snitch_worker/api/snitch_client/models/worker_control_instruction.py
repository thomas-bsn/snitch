from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerControlInstruction")



@_attrs_define
class WorkerControlInstruction:
    """
        Attributes:
            paused (bool):
     """

    paused: bool





    def to_dict(self) -> dict[str, Any]:
        paused = self.paused


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paused": paused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused = d.pop("paused")

        worker_control_instruction = cls(
            paused=paused,
        )

        return worker_control_instruction
