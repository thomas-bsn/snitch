from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerRestartInput")



@_attrs_define
class WorkerRestartInput:
    """
        Attributes:
            reason (str):
            idempotency_key (str):
     """

    reason: str
    idempotency_key: str





    def to_dict(self) -> dict[str, Any]:
        reason = self.reason

        idempotency_key = self.idempotency_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reason": reason,
            "idempotency_key": idempotency_key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reason = d.pop("reason")

        idempotency_key = d.pop("idempotency_key")

        worker_restart_input = cls(
            reason=reason,
            idempotency_key=idempotency_key,
        )

        return worker_restart_input
