from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.clip_history_step import ClipHistoryStep





T = TypeVar("T", bound="ClipHistoryItem")



@_attrs_define
class ClipHistoryItem:
    """
        Attributes:
            clip_id (int):
            original_name (str):
            last_activity_at (datetime.datetime):
            steps (list[ClipHistoryStep]):
     """

    clip_id: int
    original_name: str
    last_activity_at: datetime.datetime
    steps: list[ClipHistoryStep]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_history_step import ClipHistoryStep # noqa: PLC0415
        clip_id = self.clip_id

        original_name = self.original_name

        last_activity_at = self.last_activity_at.isoformat()

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "original_name": original_name,
            "last_activity_at": last_activity_at,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_history_step import ClipHistoryStep # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        original_name = d.pop("original_name")

        last_activity_at = datetime.datetime.fromisoformat(d.pop("last_activity_at"))




        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = ClipHistoryStep.from_dict(steps_item_data)



            steps.append(steps_item)


        clip_history_item = cls(
            clip_id=clip_id,
            original_name=original_name,
            last_activity_at=last_activity_at,
            steps=steps,
        )


        clip_history_item.additional_properties = d
        return clip_history_item

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
