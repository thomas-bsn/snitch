from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_install_event_status import WorkerInstallEventStatus
from ..models.worker_install_phase import WorkerInstallPhase
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerInstallEvent")



@_attrs_define
class WorkerInstallEvent:
    """
        Attributes:
            phase (WorkerInstallPhase):
            status (WorkerInstallEventStatus):
            cause_code (str):
            occurred_at (datetime.datetime):
            received_at (datetime.datetime):
            received_later (bool): Vrai si l'événement a été mis en file hors ligne et reçu longtemps après son horodatage
                d'origine.
     """

    phase: WorkerInstallPhase
    status: WorkerInstallEventStatus
    cause_code: str
    occurred_at: datetime.datetime
    received_at: datetime.datetime
    received_later: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        phase = self.phase.value

        status = self.status.value

        cause_code = self.cause_code

        occurred_at = self.occurred_at.isoformat()

        received_at = self.received_at.isoformat()

        received_later = self.received_later


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "phase": phase,
            "status": status,
            "cause_code": cause_code,
            "occurred_at": occurred_at,
            "received_at": received_at,
            "received_later": received_later,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phase = WorkerInstallPhase(d.pop("phase"))




        status = WorkerInstallEventStatus(d.pop("status"))




        cause_code = d.pop("cause_code")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurred_at"))




        received_at = datetime.datetime.fromisoformat(d.pop("received_at"))




        received_later = d.pop("received_later")

        worker_install_event = cls(
            phase=phase,
            status=status,
            cause_code=cause_code,
            occurred_at=occurred_at,
            received_at=received_at,
            received_later=received_later,
        )


        worker_install_event.additional_properties = d
        return worker_install_event

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
