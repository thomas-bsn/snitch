from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TaskQueueSummary")



@_attrs_define
class TaskQueueSummary:
    """ What the queue bar shows. running counts only leased tasks with an unexpired lease, which is what fills the ceiling;
    waiting counts clips queued, ready, in retry or waiting, which occupy no slot.

        Attributes:
            running (int):
            waiting (int):
            max_concurrency (int):
            paused (bool):
     """

    running: int
    waiting: int
    max_concurrency: int
    paused: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        running = self.running

        waiting = self.waiting

        max_concurrency = self.max_concurrency

        paused = self.paused


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "running": running,
            "waiting": waiting,
            "max_concurrency": max_concurrency,
            "paused": paused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        running = d.pop("running")

        waiting = d.pop("waiting")

        max_concurrency = d.pop("max_concurrency")

        paused = d.pop("paused")

        task_queue_summary = cls(
            running=running,
            waiting=waiting,
            max_concurrency=max_concurrency,
            paused=paused,
        )


        task_queue_summary.additional_properties = d
        return task_queue_summary

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
