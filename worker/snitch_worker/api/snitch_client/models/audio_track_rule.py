from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AudioTrackRule")



@_attrs_define
class AudioTrackRule:
    """
        Attributes:
            title (str | Unset):
            language (str | Unset):
            channel_layout (str | Unset):
            channels (int | Unset):
            fallback_audio_stream_index (int | Unset):
     """

    title: str | Unset = UNSET
    language: str | Unset = UNSET
    channel_layout: str | Unset = UNSET
    channels: int | Unset = UNSET
    fallback_audio_stream_index: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        title = self.title

        language = self.language

        channel_layout = self.channel_layout

        channels = self.channels

        fallback_audio_stream_index = self.fallback_audio_stream_index


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if title is not UNSET:
            field_dict["title"] = title
        if language is not UNSET:
            field_dict["language"] = language
        if channel_layout is not UNSET:
            field_dict["channel_layout"] = channel_layout
        if channels is not UNSET:
            field_dict["channels"] = channels
        if fallback_audio_stream_index is not UNSET:
            field_dict["fallback_audio_stream_index"] = fallback_audio_stream_index

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        title = d.pop("title", UNSET)

        language = d.pop("language", UNSET)

        channel_layout = d.pop("channel_layout", UNSET)

        channels = d.pop("channels", UNSET)

        fallback_audio_stream_index = d.pop("fallback_audio_stream_index", UNSET)

        audio_track_rule = cls(
            title=title,
            language=language,
            channel_layout=channel_layout,
            channels=channels,
            fallback_audio_stream_index=fallback_audio_stream_index,
        )


        audio_track_rule.additional_properties = d
        return audio_track_rule

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
