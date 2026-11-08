from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_engine_history_entry_cause_code import WorkerEngineHistoryEntryCauseCode
from ..models.worker_engine_history_entry_state import WorkerEngineHistoryEntryState
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerEngineHistoryEntry")



@_attrs_define
class WorkerEngineHistoryEntry:
    """
        Attributes:
            state (WorkerEngineHistoryEntryState): uninstall_failed : retrait refusé par le Worker, le runtime reste en
                place.
            cause_code (WorkerEngineHistoryEntryCauseCode):
            generation (int):
            ignored (bool): Rapport tardif ignoré et journalisé.
            reason (str):
            created_at (datetime.datetime):
     """

    state: WorkerEngineHistoryEntryState
    cause_code: WorkerEngineHistoryEntryCauseCode
    generation: int
    ignored: bool
    reason: str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        cause_code = self.cause_code.value

        generation = self.generation

        ignored = self.ignored

        reason = self.reason

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "state": state,
            "cause_code": cause_code,
            "generation": generation,
            "ignored": ignored,
            "reason": reason,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = WorkerEngineHistoryEntryState(d.pop("state"))




        cause_code = WorkerEngineHistoryEntryCauseCode(d.pop("cause_code"))




        generation = d.pop("generation")

        ignored = d.pop("ignored")

        reason = d.pop("reason")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        worker_engine_history_entry = cls(
            state=state,
            cause_code=cause_code,
            generation=generation,
            ignored=ignored,
            reason=reason,
            created_at=created_at,
        )


        worker_engine_history_entry.additional_properties = d
        return worker_engine_history_entry

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
