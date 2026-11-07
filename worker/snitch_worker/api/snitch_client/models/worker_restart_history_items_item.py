from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WorkerRestartHistoryItemsItem")



@_attrs_define
class WorkerRestartHistoryItemsItem:
    """
        Attributes:
            id (str):
            worker_id (str):
            actor_id (str):
            actor (str):
            reason (str):
            status (str):
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            result (str):
     """

    id: str
    worker_id: str
    actor_id: str
    actor: str
    reason: str
    status: str
    created_at: datetime.datetime
    updated_at: datetime.datetime
    result: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        worker_id = self.worker_id

        actor_id = self.actor_id

        actor = self.actor

        reason = self.reason

        status = self.status

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        result = self.result


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "worker_id": worker_id,
            "actor_id": actor_id,
            "actor": actor,
            "reason": reason,
            "status": status,
            "created_at": created_at,
            "updated_at": updated_at,
            "result": result,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        worker_id = d.pop("worker_id")

        actor_id = d.pop("actor_id")

        actor = d.pop("actor")

        reason = d.pop("reason")

        status = d.pop("status")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        result = d.pop("result")

        worker_restart_history_items_item = cls(
            id=id,
            worker_id=worker_id,
            actor_id=actor_id,
            actor=actor,
            reason=reason,
            status=status,
            created_at=created_at,
            updated_at=updated_at,
            result=result,
        )


        worker_restart_history_items_item.additional_properties = d
        return worker_restart_history_items_item

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
