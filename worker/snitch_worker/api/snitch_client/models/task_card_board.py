from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.task_card import TaskCard
  from ..models.task_queue_summary import TaskQueueSummary





T = TypeVar("T", bound="TaskCardBoard")



@_attrs_define
class TaskCardBoard:
    """
        Attributes:
            queue (TaskQueueSummary): What the queue bar shows. running counts only leased tasks with an unexpired lease,
                which is what fills the ceiling; waiting counts clips queued, ready, in retry or waiting, which occupy no slot.
            cards (list[TaskCard]):
            completed (list[TaskCard]):
     """

    queue: TaskQueueSummary
    cards: list[TaskCard]
    completed: list[TaskCard]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.task_card import TaskCard # noqa: PLC0415
        from ..models.task_queue_summary import TaskQueueSummary # noqa: PLC0415
        queue = self.queue.to_dict()

        cards = []
        for cards_item_data in self.cards:
            cards_item = cards_item_data.to_dict()
            cards.append(cards_item)



        completed = []
        for completed_item_data in self.completed:
            completed_item = completed_item_data.to_dict()
            completed.append(completed_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "queue": queue,
            "cards": cards,
            "completed": completed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.task_card import TaskCard # noqa: PLC0415
        from ..models.task_queue_summary import TaskQueueSummary # noqa: PLC0415
        d = dict(src_dict)
        queue = TaskQueueSummary.from_dict(d.pop("queue"))




        cards = []
        _cards = d.pop("cards")
        for cards_item_data in (_cards):
            cards_item = TaskCard.from_dict(cards_item_data)



            cards.append(cards_item)


        completed = []
        _completed = d.pop("completed")
        for completed_item_data in (_completed):
            completed_item = TaskCard.from_dict(completed_item_data)



            completed.append(completed_item)


        task_card_board = cls(
            queue=queue,
            cards=cards,
            completed=completed,
        )


        task_card_board.additional_properties = d
        return task_card_board

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
