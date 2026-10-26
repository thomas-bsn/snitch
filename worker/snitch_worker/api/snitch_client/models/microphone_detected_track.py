from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="MicrophoneDetectedTrack")



@_attrs_define
class MicrophoneDetectedTrack:
    """
        Attributes:
            audio_stream_index (int):
            title (str):
            language (str):
            channels (int):
            channel_layout (str):
            clip_count (int): Nombre de clips prêts de la source portant cette piste.
     """

    audio_stream_index: int
    title: str
    language: str
    channels: int
    channel_layout: str
    clip_count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        audio_stream_index = self.audio_stream_index

        title = self.title

        language = self.language

        channels = self.channels

        channel_layout = self.channel_layout

        clip_count = self.clip_count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "audio_stream_index": audio_stream_index,
            "title": title,
            "language": language,
            "channels": channels,
            "channel_layout": channel_layout,
            "clip_count": clip_count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audio_stream_index = d.pop("audio_stream_index")

        title = d.pop("title")

        language = d.pop("language")

        channels = d.pop("channels")

        channel_layout = d.pop("channel_layout")

        clip_count = d.pop("clip_count")

        microphone_detected_track = cls(
            audio_stream_index=audio_stream_index,
            title=title,
            language=language,
            channels=channels,
            channel_layout=channel_layout,
            clip_count=clip_count,
        )


        microphone_detected_track.additional_properties = d
        return microphone_detected_track

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
