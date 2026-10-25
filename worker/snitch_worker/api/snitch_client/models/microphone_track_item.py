from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.microphone_track_item_role import MicrophoneTrackItemRole






T = TypeVar("T", bound="MicrophoneTrackItem")



@_attrs_define
class MicrophoneTrackItem:
    """
        Attributes:
            index (int):
            title (str):
            language (str):
            channels (int):
            channel_layout (str):
            role (MicrophoneTrackItemRole): other : piste traitée normalement (diarisation).
     """

    index: int
    title: str
    language: str
    channels: int
    channel_layout: str
    role: MicrophoneTrackItemRole
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        title = self.title

        language = self.language

        channels = self.channels

        channel_layout = self.channel_layout

        role = self.role.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "index": index,
            "title": title,
            "language": language,
            "channels": channels,
            "channel_layout": channel_layout,
            "role": role,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        title = d.pop("title")

        language = d.pop("language")

        channels = d.pop("channels")

        channel_layout = d.pop("channel_layout")

        role = MicrophoneTrackItemRole(d.pop("role"))




        microphone_track_item = cls(
            index=index,
            title=title,
            language=language,
            channels=channels,
            channel_layout=channel_layout,
            role=role,
        )


        microphone_track_item.additional_properties = d
        return microphone_track_item

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
