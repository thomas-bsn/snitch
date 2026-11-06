from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_install_event_input import WorkerInstallEventInput





T = TypeVar("T", bound="PostWorkerInstallEventsBody")



@_attrs_define
class PostWorkerInstallEventsBody:
    """
        Attributes:
            events (list[WorkerInstallEventInput]):
     """

    events: list[WorkerInstallEventInput]





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_install_event_input import WorkerInstallEventInput # noqa: PLC0415
        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_install_event_input import WorkerInstallEventInput # noqa: PLC0415
        d = dict(src_dict)
        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = WorkerInstallEventInput.from_dict(events_item_data)



            events.append(events_item)


        post_worker_install_events_body = cls(
            events=events,
        )

        return post_worker_install_events_body
