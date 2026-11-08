from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_engine_status_cause_code import WorkerEngineStatusCauseCode
from ..models.worker_engine_status_state import WorkerEngineStatusState
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerEngineStatus")



@_attrs_define
class WorkerEngineStatus:
    """
        Attributes:
            order_id (str):
            digest (str):
            state (WorkerEngineStatusState):
            cause_code (WorkerEngineStatusCauseCode):
            bytes_received (int | Unset):
            bytes_total (int | Unset):
            local_identity (str | Unset): Identité canonique locale du runtime sur le volume du Worker (obligatoire pour
                ready) ; distincte de l'identité signée.
     """

    order_id: str
    digest: str
    state: WorkerEngineStatusState
    cause_code: WorkerEngineStatusCauseCode
    bytes_received: int | Unset = UNSET
    bytes_total: int | Unset = UNSET
    local_identity: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        order_id = self.order_id

        digest = self.digest

        state = self.state.value

        cause_code = self.cause_code.value

        bytes_received = self.bytes_received

        bytes_total = self.bytes_total

        local_identity = self.local_identity


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "order_id": order_id,
            "digest": digest,
            "state": state,
            "cause_code": cause_code,
        })
        if bytes_received is not UNSET:
            field_dict["bytes_received"] = bytes_received
        if bytes_total is not UNSET:
            field_dict["bytes_total"] = bytes_total
        if local_identity is not UNSET:
            field_dict["local_identity"] = local_identity

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        order_id = d.pop("order_id")

        digest = d.pop("digest")

        state = WorkerEngineStatusState(d.pop("state"))




        cause_code = WorkerEngineStatusCauseCode(d.pop("cause_code"))




        bytes_received = d.pop("bytes_received", UNSET)

        bytes_total = d.pop("bytes_total", UNSET)

        local_identity = d.pop("local_identity", UNSET)

        worker_engine_status = cls(
            order_id=order_id,
            digest=digest,
            state=state,
            cause_code=cause_code,
            bytes_received=bytes_received,
            bytes_total=bytes_total,
            local_identity=local_identity,
        )

        return worker_engine_status
