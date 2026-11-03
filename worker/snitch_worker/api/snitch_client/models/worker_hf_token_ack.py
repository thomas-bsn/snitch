from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_hf_token_ack_error_code import WorkerHFTokenAckErrorCode
from ..models.worker_hf_token_ack_outcome import WorkerHFTokenAckOutcome
from ..models.worker_hf_token_ack_source import WorkerHFTokenAckSource
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerHFTokenAck")



@_attrs_define
class WorkerHFTokenAck:
    """
        Attributes:
            version (str):
            outcome (WorkerHFTokenAckOutcome):
            source (WorkerHFTokenAckSource):
            error_code (WorkerHFTokenAckErrorCode | Unset):
     """

    version: str
    outcome: WorkerHFTokenAckOutcome
    source: WorkerHFTokenAckSource
    error_code: WorkerHFTokenAckErrorCode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        outcome = self.outcome.value

        source = self.source.value

        error_code: str | Unset = UNSET
        if not isinstance(self.error_code, Unset):
            error_code = self.error_code.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
            "outcome": outcome,
            "source": source,
        })
        if error_code is not UNSET:
            field_dict["error_code"] = error_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        outcome = WorkerHFTokenAckOutcome(d.pop("outcome"))




        source = WorkerHFTokenAckSource(d.pop("source"))




        _error_code = d.pop("error_code", UNSET)
        error_code: WorkerHFTokenAckErrorCode | Unset
        if isinstance(_error_code,  Unset):
            error_code = UNSET
        else:
            error_code = WorkerHFTokenAckErrorCode(_error_code)




        worker_hf_token_ack = cls(
            version=version,
            outcome=outcome,
            source=source,
            error_code=error_code,
        )

        return worker_hf_token_ack
