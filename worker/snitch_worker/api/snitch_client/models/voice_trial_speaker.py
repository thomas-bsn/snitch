from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceTrialSpeaker")



@_attrs_define
class VoiceTrialSpeaker:
    """
        Attributes:
            label (str):
            speaking_seconds (float):
            turn_count (int):
     """

    label: str
    speaking_seconds: float
    turn_count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        speaking_seconds = self.speaking_seconds

        turn_count = self.turn_count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "label": label,
            "speaking_seconds": speaking_seconds,
            "turn_count": turn_count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        speaking_seconds = d.pop("speaking_seconds")

        turn_count = d.pop("turn_count")

        voice_trial_speaker = cls(
            label=label,
            speaking_seconds=speaking_seconds,
            turn_count=turn_count,
        )


        voice_trial_speaker.additional_properties = d
        return voice_trial_speaker

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
