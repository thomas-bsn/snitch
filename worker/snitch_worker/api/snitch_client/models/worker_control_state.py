from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WorkerControlState")



@_attrs_define
class WorkerControlState:
    """
        Attributes:
            paused (bool): Pause demandée.
            applied_paused (bool | None): Pause confirmée par le Worker ; null tant qu'il n'a rien confirmé. Diffère de
                `paused` pendant que la consigne est en route.
     """

    paused: bool
    applied_paused: bool | None





    def to_dict(self) -> dict[str, Any]:
        paused = self.paused

        applied_paused: bool | None
        applied_paused = self.applied_paused


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "paused": paused,
            "applied_paused": applied_paused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused = d.pop("paused")

        def _parse_applied_paused(data: object) -> bool | None:
            if data is None:
                return data
            return cast(bool | None, data)

        applied_paused = _parse_applied_paused(d.pop("applied_paused"))


        worker_control_state = cls(
            paused=paused,
            applied_paused=applied_paused,
        )

        return worker_control_state
