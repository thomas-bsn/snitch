from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.segment_correction import SegmentCorrection
  from ..models.segment_history_entry import SegmentHistoryEntry





T = TypeVar("T", bound="SegmentHistory")



@_attrs_define
class SegmentHistory:
    """
        Attributes:
            transcription_job_id (int):
            index (int):
            original_text (str):
            current (SegmentCorrection):
            items (list[SegmentHistoryEntry]):
     """

    transcription_job_id: int
    index: int
    original_text: str
    current: SegmentCorrection
    items: list[SegmentHistoryEntry]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_correction import SegmentCorrection # noqa: PLC0415
        from ..models.segment_history_entry import SegmentHistoryEntry # noqa: PLC0415
        transcription_job_id = self.transcription_job_id

        index = self.index

        original_text = self.original_text

        current = self.current.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "transcription_job_id": transcription_job_id,
            "index": index,
            "original_text": original_text,
            "current": current,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_correction import SegmentCorrection # noqa: PLC0415
        from ..models.segment_history_entry import SegmentHistoryEntry # noqa: PLC0415
        d = dict(src_dict)
        transcription_job_id = d.pop("transcription_job_id")

        index = d.pop("index")

        original_text = d.pop("original_text")

        current = SegmentCorrection.from_dict(d.pop("current"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SegmentHistoryEntry.from_dict(items_item_data)



            items.append(items_item)


        segment_history = cls(
            transcription_job_id=transcription_job_id,
            index=index,
            original_text=original_text,
            current=current,
            items=items,
        )


        segment_history.additional_properties = d
        return segment_history

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
