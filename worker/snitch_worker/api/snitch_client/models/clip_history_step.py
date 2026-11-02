from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.job_event import JobEvent





T = TypeVar("T", bound="ClipHistoryStep")



@_attrs_define
class ClipHistoryStep:
    """
        Attributes:
            job_id (int):
            step (str):
            status (str):
            attempts (int):
            max_attempts (int):
            cycle (int):
            worker (None | str):
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            events (list[JobEvent]):
     """

    job_id: int
    step: str
    status: str
    attempts: int
    max_attempts: int
    cycle: int
    worker: None | str
    created_at: datetime.datetime
    updated_at: datetime.datetime
    events: list[JobEvent]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job_event import JobEvent # noqa: PLC0415
        job_id = self.job_id

        step = self.step

        status = self.status

        attempts = self.attempts

        max_attempts = self.max_attempts

        cycle = self.cycle

        worker: None | str
        worker = self.worker

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "step": step,
            "status": status,
            "attempts": attempts,
            "max_attempts": max_attempts,
            "cycle": cycle,
            "worker": worker,
            "created_at": created_at,
            "updated_at": updated_at,
            "events": events,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_event import JobEvent # noqa: PLC0415
        d = dict(src_dict)
        job_id = d.pop("job_id")

        step = d.pop("step")

        status = d.pop("status")

        attempts = d.pop("attempts")

        max_attempts = d.pop("max_attempts")

        cycle = d.pop("cycle")

        def _parse_worker(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        worker = _parse_worker(d.pop("worker"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = JobEvent.from_dict(events_item_data)



            events.append(events_item)


        clip_history_step = cls(
            job_id=job_id,
            step=step,
            status=status,
            attempts=attempts,
            max_attempts=max_attempts,
            cycle=cycle,
            worker=worker,
            created_at=created_at,
            updated_at=updated_at,
            events=events,
        )


        clip_history_step.additional_properties = d
        return clip_history_step

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
