from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="MediaTrack")



@_attrs_define
class MediaTrack:
    """
        Attributes:
            index (int):
            type_ (str):
            codec (str):
            title (str | Unset):
            language (str | Unset):
            width (int | Unset):
            height (int | Unset):
            channels (int | Unset):
            channel_layout (str | Unset):
            duration_seconds (float | None | Unset): Durée propre au flux ffprobe ; null si inconnue. Jamais la durée du
                conteneur.
            start_seconds (float | None | Unset): Décalage de départ propre au flux ffprobe ; null si inconnu.
     """

    index: int
    type_: str
    codec: str
    title: str | Unset = UNSET
    language: str | Unset = UNSET
    width: int | Unset = UNSET
    height: int | Unset = UNSET
    channels: int | Unset = UNSET
    channel_layout: str | Unset = UNSET
    duration_seconds: float | None | Unset = UNSET
    start_seconds: float | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        type_ = self.type_

        codec = self.codec

        title = self.title

        language = self.language

        width = self.width

        height = self.height

        channels = self.channels

        channel_layout = self.channel_layout

        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds

        start_seconds: float | None | Unset
        if isinstance(self.start_seconds, Unset):
            start_seconds = UNSET
        else:
            start_seconds = self.start_seconds


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "index": index,
            "type": type_,
            "codec": codec,
        })
        if title is not UNSET:
            field_dict["title"] = title
        if language is not UNSET:
            field_dict["language"] = language
        if width is not UNSET:
            field_dict["width"] = width
        if height is not UNSET:
            field_dict["height"] = height
        if channels is not UNSET:
            field_dict["channels"] = channels
        if channel_layout is not UNSET:
            field_dict["channel_layout"] = channel_layout
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if start_seconds is not UNSET:
            field_dict["start_seconds"] = start_seconds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        type_ = d.pop("type")

        codec = d.pop("codec")

        title = d.pop("title", UNSET)

        language = d.pop("language", UNSET)

        width = d.pop("width", UNSET)

        height = d.pop("height", UNSET)

        channels = d.pop("channels", UNSET)

        channel_layout = d.pop("channel_layout", UNSET)

        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        def _parse_start_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        start_seconds = _parse_start_seconds(d.pop("start_seconds", UNSET))


        media_track = cls(
            index=index,
            type_=type_,
            codec=codec,
            title=title,
            language=language,
            width=width,
            height=height,
            channels=channels,
            channel_layout=channel_layout,
            duration_seconds=duration_seconds,
            start_seconds=start_seconds,
        )


        media_track.additional_properties = d
        return media_track

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
