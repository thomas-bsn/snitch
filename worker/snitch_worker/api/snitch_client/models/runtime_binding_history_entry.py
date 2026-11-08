from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.runtime_binding_history_entry_intent import RuntimeBindingHistoryEntryIntent
from ..models.runtime_binding_history_entry_state import RuntimeBindingHistoryEntryState
from typing import cast
import datetime






T = TypeVar("T", bound="RuntimeBindingHistoryEntry")



@_attrs_define
class RuntimeBindingHistoryEntry:
    """
        Attributes:
            intent (RuntimeBindingHistoryEntryIntent):
            digest (str):
            version (str):
            state (RuntimeBindingHistoryEntryState):
            cause_code (str):
            ignored (bool): Rapport tardif journalisé sans effet.
            reason (str):
            created_at (datetime.datetime):
     """

    intent: RuntimeBindingHistoryEntryIntent
    digest: str
    version: str
    state: RuntimeBindingHistoryEntryState
    cause_code: str
    ignored: bool
    reason: str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        intent = self.intent.value

        digest = self.digest

        version = self.version

        state = self.state.value

        cause_code = self.cause_code

        ignored = self.ignored

        reason = self.reason

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "intent": intent,
            "digest": digest,
            "version": version,
            "state": state,
            "cause_code": cause_code,
            "ignored": ignored,
            "reason": reason,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        intent = RuntimeBindingHistoryEntryIntent(d.pop("intent"))




        digest = d.pop("digest")

        version = d.pop("version")

        state = RuntimeBindingHistoryEntryState(d.pop("state"))




        cause_code = d.pop("cause_code")

        ignored = d.pop("ignored")

        reason = d.pop("reason")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        runtime_binding_history_entry = cls(
            intent=intent,
            digest=digest,
            version=version,
            state=state,
            cause_code=cause_code,
            ignored=ignored,
            reason=reason,
            created_at=created_at,
        )


        runtime_binding_history_entry.additional_properties = d
        return runtime_binding_history_entry

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
