from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_voice_model_status_cause_code import WorkerVoiceModelStatusCauseCode
from ..models.worker_voice_model_status_state import WorkerVoiceModelStatusState
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerVoiceModelStatus")



@_attrs_define
class WorkerVoiceModelStatus:
    """
        Attributes:
            model_id (str):
            state (WorkerVoiceModelStatusState):
            order_id (str | Unset):
            revision (str | Unset):
            cause_code (WorkerVoiceModelStatusCauseCode | Unset):
            bytes_received (int | Unset):
            bytes_total (int | Unset):
            occupied_bytes (int | Unset):
     """

    model_id: str
    state: WorkerVoiceModelStatusState
    order_id: str | Unset = UNSET
    revision: str | Unset = UNSET
    cause_code: WorkerVoiceModelStatusCauseCode | Unset = UNSET
    bytes_received: int | Unset = UNSET
    bytes_total: int | Unset = UNSET
    occupied_bytes: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        state = self.state.value

        order_id = self.order_id

        revision = self.revision

        cause_code: str | Unset = UNSET
        if not isinstance(self.cause_code, Unset):
            cause_code = self.cause_code.value


        bytes_received = self.bytes_received

        bytes_total = self.bytes_total

        occupied_bytes = self.occupied_bytes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model_id": model_id,
            "state": state,
        })
        if order_id is not UNSET:
            field_dict["order_id"] = order_id
        if revision is not UNSET:
            field_dict["revision"] = revision
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code
        if bytes_received is not UNSET:
            field_dict["bytes_received"] = bytes_received
        if bytes_total is not UNSET:
            field_dict["bytes_total"] = bytes_total
        if occupied_bytes is not UNSET:
            field_dict["occupied_bytes"] = occupied_bytes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        state = WorkerVoiceModelStatusState(d.pop("state"))




        order_id = d.pop("order_id", UNSET)

        revision = d.pop("revision", UNSET)

        _cause_code = d.pop("cause_code", UNSET)
        cause_code: WorkerVoiceModelStatusCauseCode | Unset
        if isinstance(_cause_code,  Unset):
            cause_code = UNSET
        else:
            cause_code = WorkerVoiceModelStatusCauseCode(_cause_code)




        bytes_received = d.pop("bytes_received", UNSET)

        bytes_total = d.pop("bytes_total", UNSET)

        occupied_bytes = d.pop("occupied_bytes", UNSET)

        worker_voice_model_status = cls(
            model_id=model_id,
            state=state,
            order_id=order_id,
            revision=revision,
            cause_code=cause_code,
            bytes_received=bytes_received,
            bytes_total=bytes_total,
            occupied_bytes=occupied_bytes,
        )

        return worker_voice_model_status
