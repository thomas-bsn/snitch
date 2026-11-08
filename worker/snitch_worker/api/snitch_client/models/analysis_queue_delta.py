from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.analysis_queue_run import AnalysisQueueRun
  from ..models.analysis_queue_summary import AnalysisQueueSummary
  from ..models.analysis_queue_task import AnalysisQueueTask





T = TypeVar("T", bound="AnalysisQueueDelta")



@_attrs_define
class AnalysisQueueDelta:
    """ Ce qui a changé dans la file depuis l'état précédent du flux : l'ordre des exécutions actives, le contenu de celles
    qui sont nouvelles ou modifiées, le reste en entier.

        Attributes:
            active_ids (list[int]): Toutes les exécutions actives, dans l'ordre de la file.
            active (list[AnalysisQueueRun]): Les exécutions de active_ids nouvelles ou modifiées depuis l'état précédent.
            awaiting_review (list[AnalysisQueueRun]):
            completed (list[AnalysisQueueRun]):
            tasks (list[AnalysisQueueTask]):
            summary (AnalysisQueueSummary): Charge de la file. running_tasks suit la règle de l'ordonnanceur : seuls les
                jobs leased au bail non expiré occupent un emplacement.
     """

    active_ids: list[int]
    active: list[AnalysisQueueRun]
    awaiting_review: list[AnalysisQueueRun]
    completed: list[AnalysisQueueRun]
    tasks: list[AnalysisQueueTask]
    summary: AnalysisQueueSummary
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        from ..models.analysis_queue_summary import AnalysisQueueSummary # noqa: PLC0415
        from ..models.analysis_queue_task import AnalysisQueueTask # noqa: PLC0415
        active_ids = self.active_ids



        active = []
        for active_item_data in self.active:
            active_item = active_item_data.to_dict()
            active.append(active_item)



        awaiting_review = []
        for awaiting_review_item_data in self.awaiting_review:
            awaiting_review_item = awaiting_review_item_data.to_dict()
            awaiting_review.append(awaiting_review_item)



        completed = []
        for completed_item_data in self.completed:
            completed_item = completed_item_data.to_dict()
            completed.append(completed_item)



        tasks = []
        for tasks_item_data in self.tasks:
            tasks_item = tasks_item_data.to_dict()
            tasks.append(tasks_item)



        summary = self.summary.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "active_ids": active_ids,
            "active": active,
            "awaiting_review": awaiting_review,
            "completed": completed,
            "tasks": tasks,
            "summary": summary,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        from ..models.analysis_queue_summary import AnalysisQueueSummary # noqa: PLC0415
        from ..models.analysis_queue_task import AnalysisQueueTask # noqa: PLC0415
        d = dict(src_dict)
        active_ids = cast(list[int], d.pop("active_ids"))


        active = []
        _active = d.pop("active")
        for active_item_data in (_active):
            active_item = AnalysisQueueRun.from_dict(active_item_data)



            active.append(active_item)


        awaiting_review = []
        _awaiting_review = d.pop("awaiting_review")
        for awaiting_review_item_data in (_awaiting_review):
            awaiting_review_item = AnalysisQueueRun.from_dict(awaiting_review_item_data)



            awaiting_review.append(awaiting_review_item)


        completed = []
        _completed = d.pop("completed")
        for completed_item_data in (_completed):
            completed_item = AnalysisQueueRun.from_dict(completed_item_data)



            completed.append(completed_item)


        tasks = []
        _tasks = d.pop("tasks")
        for tasks_item_data in (_tasks):
            tasks_item = AnalysisQueueTask.from_dict(tasks_item_data)



            tasks.append(tasks_item)


        summary = AnalysisQueueSummary.from_dict(d.pop("summary"))




        analysis_queue_delta = cls(
            active_ids=active_ids,
            active=active,
            awaiting_review=awaiting_review,
            completed=completed,
            tasks=tasks,
            summary=summary,
        )


        analysis_queue_delta.additional_properties = d
        return analysis_queue_delta

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
