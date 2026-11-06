from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceTraceTurnsItem")



@_attrs_define
class VoiceTraceTurnsItem:
    """
        Attributes:
            local_speaker_id (int):
            label (str):
            start (float):
            end (float):
     """

    local_speaker_id: int
    label: str
    start: float
    end: float
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        local_speaker_id = self.local_speaker_id

        label = self.label

        start = self.start

        end = self.end


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "local_speaker_id": local_speaker_id,
            "label": label,
            "start": start,
            "end": end,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        local_speaker_id = d.pop("local_speaker_id")

        label = d.pop("label")

        start = d.pop("start")

        end = d.pop("end")

        voice_trace_turns_item = cls(
            local_speaker_id=local_speaker_id,
            label=label,
            start=start,
            end=end,
        )


        voice_trace_turns_item.additional_properties = d
        return voice_trace_turns_item

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
