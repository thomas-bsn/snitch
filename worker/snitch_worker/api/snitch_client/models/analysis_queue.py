from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.analysis_queue_run import AnalysisQueueRun
  from ..models.analysis_queue_task import AnalysisQueueTask





T = TypeVar("T", bound="AnalysisQueue")



@_attrs_define
class AnalysisQueue:
    """
        Attributes:
            active (list[AnalysisQueueRun]):
            completed (list[AnalysisQueueRun]):
            tasks (list[AnalysisQueueTask]):
     """

    active: list[AnalysisQueueRun]
    completed: list[AnalysisQueueRun]
    tasks: list[AnalysisQueueTask]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        from ..models.analysis_queue_task import AnalysisQueueTask # noqa: PLC0415
        active = []
        for active_item_data in self.active:
            active_item = active_item_data.to_dict()
            active.append(active_item)



        completed = []
        for completed_item_data in self.completed:
            completed_item = completed_item_data.to_dict()
            completed.append(completed_item)



        tasks = []
        for tasks_item_data in self.tasks:
            tasks_item = tasks_item_data.to_dict()
            tasks.append(tasks_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "active": active,
            "completed": completed,
            "tasks": tasks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.analysis_queue_run import AnalysisQueueRun # noqa: PLC0415
        from ..models.analysis_queue_task import AnalysisQueueTask # noqa: PLC0415
        d = dict(src_dict)
        active = []
        _active = d.pop("active")
        for active_item_data in (_active):
            active_item = AnalysisQueueRun.from_dict(active_item_data)



            active.append(active_item)


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


        analysis_queue = cls(
            active=active,
            completed=completed,
            tasks=tasks,
        )


        analysis_queue.additional_properties = d
        return analysis_queue

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
