from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.job_event_actor import JobEventActor
from ..models.job_event_event import JobEventEvent
from typing import cast
import datetime






T = TypeVar("T", bound="JobEvent")



@_attrs_define
class JobEvent:
    """
        Attributes:
            id (int):
            event (JobEventEvent):
            cycle (int):
            attempt (int | None):
            actor (JobEventActor):
            actor_id (str):
            error_code (str):
            created_at (datetime.datetime):
     """

    id: int
    event: JobEventEvent
    cycle: int
    attempt: int | None
    actor: JobEventActor
    actor_id: str
    error_code: str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        event = self.event.value

        cycle = self.cycle

        attempt: int | None
        attempt = self.attempt

        actor = self.actor.value

        actor_id = self.actor_id

        error_code = self.error_code

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "event": event,
            "cycle": cycle,
            "attempt": attempt,
            "actor": actor,
            "actor_id": actor_id,
            "error_code": error_code,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        event = JobEventEvent(d.pop("event"))




        cycle = d.pop("cycle")

        def _parse_attempt(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        attempt = _parse_attempt(d.pop("attempt"))


        actor = JobEventActor(d.pop("actor"))




        actor_id = d.pop("actor_id")

        error_code = d.pop("error_code")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        job_event = cls(
            id=id,
            event=event,
            cycle=cycle,
            attempt=attempt,
            actor=actor,
            actor_id=actor_id,
            error_code=error_code,
            created_at=created_at,
        )


        job_event.additional_properties = d
        return job_event

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
