from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AudioTrackInventory")



@_attrs_define
class AudioTrackInventory:
    """
        Attributes:
            index (int):
            title (str):
            codec (str):
            language (str):
            channels (int):
            channel_layout (str):
            duration_seconds (float | None):
            start_seconds (float):
            evidence (list[str]):
     """

    index: int
    title: str
    codec: str
    language: str
    channels: int
    channel_layout: str
    duration_seconds: float | None
    start_seconds: float
    evidence: list[str]





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        title = self.title

        codec = self.codec

        language = self.language

        channels = self.channels

        channel_layout = self.channel_layout

        duration_seconds: float | None
        duration_seconds = self.duration_seconds

        start_seconds = self.start_seconds

        evidence = self.evidence




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "title": title,
            "codec": codec,
            "language": language,
            "channels": channels,
            "channel_layout": channel_layout,
            "duration_seconds": duration_seconds,
            "start_seconds": start_seconds,
            "evidence": evidence,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        title = d.pop("title")

        codec = d.pop("codec")

        language = d.pop("language")

        channels = d.pop("channels")

        channel_layout = d.pop("channel_layout")

        def _parse_duration_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds"))


        start_seconds = d.pop("start_seconds")

        evidence = cast(list[str], d.pop("evidence"))


        audio_track_inventory = cls(
            index=index,
            title=title,
            codec=codec,
            language=language,
            channels=channels,
            channel_layout=channel_layout,
            duration_seconds=duration_seconds,
            start_seconds=start_seconds,
            evidence=evidence,
        )

        return audio_track_inventory
