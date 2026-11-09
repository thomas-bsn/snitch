from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_control_input_action import WorkerControlInputAction






T = TypeVar("T", bound="WorkerControlInput")



@_attrs_define
class WorkerControlInput:
    """
        Attributes:
            action (WorkerControlInputAction):
     """

    action: WorkerControlInputAction





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = WorkerControlInputAction(d.pop("action"))




        worker_control_input = cls(
            action=action,
        )

        return worker_control_input
