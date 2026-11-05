from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_model_order_ack_cause_code import WorkerModelOrderAckCauseCode
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerModelOrderAck")



@_attrs_define
class WorkerModelOrderAck:
    """
        Attributes:
            accepted (bool):
            cause_code (WorkerModelOrderAckCauseCode | Unset):
     """

    accepted: bool
    cause_code: WorkerModelOrderAckCauseCode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        cause_code: str | Unset = UNSET
        if not isinstance(self.cause_code, Unset):
            cause_code = self.cause_code.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
        })
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        _cause_code = d.pop("cause_code", UNSET)
        cause_code: WorkerModelOrderAckCauseCode | Unset
        if isinstance(_cause_code,  Unset):
            cause_code = UNSET
        else:
            cause_code = WorkerModelOrderAckCauseCode(_cause_code)




        worker_model_order_ack = cls(
            accepted=accepted,
            cause_code=cause_code,
        )

        return worker_model_order_ack
