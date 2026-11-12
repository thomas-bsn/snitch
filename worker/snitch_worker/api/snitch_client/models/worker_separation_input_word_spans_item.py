from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerSeparationInputWordSpansItem")



@_attrs_define
class WorkerSeparationInputWordSpansItem:
    """
        Attributes:
            speaker (str):
            start (float):
            end (float):
     """

    speaker: str
    start: float
    end: float
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        speaker = self.speaker

        start = self.start

        end = self.end


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "speaker": speaker,
            "start": start,
            "end": end,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        speaker = d.pop("speaker")

        start = d.pop("start")

        end = d.pop("end")

        worker_separation_input_word_spans_item = cls(
            speaker=speaker,
            start=start,
            end=end,
        )


        worker_separation_input_word_spans_item.additional_properties = d
        return worker_separation_input_word_spans_item

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
