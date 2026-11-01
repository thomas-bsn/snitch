from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_install_event_input_status import WorkerInstallEventInputStatus
from ..models.worker_install_phase import WorkerInstallPhase
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerInstallEventInput")



@_attrs_define
class WorkerInstallEventInput:
    """
        Attributes:
            phase (WorkerInstallPhase):
            status (WorkerInstallEventInputStatus):
            occurred_at (datetime.datetime): Horodatage d'origine, conservé même si l'événement a été mis en file hors
                ligne.
            cause_code (str | Unset): Obligatoire (un code de WorkerInstallCause) si le statut est failed, vide sinon.
     """

    phase: WorkerInstallPhase
    status: WorkerInstallEventInputStatus
    occurred_at: datetime.datetime
    cause_code: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phase = self.phase.value

        status = self.status.value

        occurred_at = self.occurred_at.isoformat()

        cause_code = self.cause_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "phase": phase,
            "status": status,
            "occurred_at": occurred_at,
        })
        if cause_code is not UNSET:
            field_dict["cause_code"] = cause_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phase = WorkerInstallPhase(d.pop("phase"))




        status = WorkerInstallEventInputStatus(d.pop("status"))




        occurred_at = datetime.datetime.fromisoformat(d.pop("occurred_at"))




        cause_code = d.pop("cause_code", UNSET)

        worker_install_event_input = cls(
            phase=phase,
            status=status,
            occurred_at=occurred_at,
            cause_code=cause_code,
        )

        return worker_install_event_input
