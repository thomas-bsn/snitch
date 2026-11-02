from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AnalysisQueueSummary")



@_attrs_define
class AnalysisQueueSummary:
    """ Charge de la file. running_tasks suit la règle de l'ordonnanceur : seuls les jobs leased au bail non expiré occupent
    un emplacement.

        Attributes:
            running_tasks (int):
            waiting_clips (int): Clips dont l'analyse est active et qui n'occupent aucun emplacement (attente, reprise
                différée ou suspendus).
            max_concurrency (int):
            paused (bool):
     """

    running_tasks: int
    waiting_clips: int
    max_concurrency: int
    paused: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        running_tasks = self.running_tasks

        waiting_clips = self.waiting_clips

        max_concurrency = self.max_concurrency

        paused = self.paused


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "running_tasks": running_tasks,
            "waiting_clips": waiting_clips,
            "max_concurrency": max_concurrency,
            "paused": paused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        running_tasks = d.pop("running_tasks")

        waiting_clips = d.pop("waiting_clips")

        max_concurrency = d.pop("max_concurrency")

        paused = d.pop("paused")

        analysis_queue_summary = cls(
            running_tasks=running_tasks,
            waiting_clips=waiting_clips,
            max_concurrency=max_concurrency,
            paused=paused,
        )


        analysis_queue_summary.additional_properties = d
        return analysis_queue_summary

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
