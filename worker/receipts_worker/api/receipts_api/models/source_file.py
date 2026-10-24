from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.source_file_status import SourceFileStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.media_track import MediaTrack





T = TypeVar("T", bound="SourceFile")



@_attrs_define
class SourceFile:
    """
        Attributes:
            name (str):
            relative_path (str):
            status (SourceFileStatus):
            size_bytes (int):
            modified_at (datetime.datetime):
            recorded_at_estimated (bool):
            tracks (list[MediaTrack]):
            error_code (str | Unset):
            discovered_at (datetime.datetime | Unset):
            recorded_at (datetime.datetime | Unset):
            duration_seconds (float | Unset):
            format_name (str | Unset):
            video_codec (str | Unset):
            audio_codec (str | Unset):
     """

    name: str
    relative_path: str
    status: SourceFileStatus
    size_bytes: int
    modified_at: datetime.datetime
    recorded_at_estimated: bool
    tracks: list[MediaTrack]
    error_code: str | Unset = UNSET
    discovered_at: datetime.datetime | Unset = UNSET
    recorded_at: datetime.datetime | Unset = UNSET
    duration_seconds: float | Unset = UNSET
    format_name: str | Unset = UNSET
    video_codec: str | Unset = UNSET
    audio_codec: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.media_track import MediaTrack # noqa: PLC0415
        name = self.name

        relative_path = self.relative_path

        status = self.status.value

        size_bytes = self.size_bytes

        modified_at = self.modified_at.isoformat()

        recorded_at_estimated = self.recorded_at_estimated

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        error_code = self.error_code

        discovered_at: str | Unset = UNSET
        if not isinstance(self.discovered_at, Unset):
            discovered_at = self.discovered_at.isoformat()

        recorded_at: str | Unset = UNSET
        if not isinstance(self.recorded_at, Unset):
            recorded_at = self.recorded_at.isoformat()

        duration_seconds = self.duration_seconds

        format_name = self.format_name

        video_codec = self.video_codec

        audio_codec = self.audio_codec


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "name": name,
            "relative_path": relative_path,
            "status": status,
            "size_bytes": size_bytes,
            "modified_at": modified_at,
            "recorded_at_estimated": recorded_at_estimated,
            "tracks": tracks,
        })
        if error_code is not UNSET:
            field_dict["error_code"] = error_code
        if discovered_at is not UNSET:
            field_dict["discovered_at"] = discovered_at
        if recorded_at is not UNSET:
            field_dict["recorded_at"] = recorded_at
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if format_name is not UNSET:
            field_dict["format_name"] = format_name
        if video_codec is not UNSET:
            field_dict["video_codec"] = video_codec
        if audio_codec is not UNSET:
            field_dict["audio_codec"] = audio_codec

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.media_track import MediaTrack # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        relative_path = d.pop("relative_path")

        status = SourceFileStatus(d.pop("status"))




        size_bytes = d.pop("size_bytes")

        modified_at = datetime.datetime.fromisoformat(d.pop("modified_at"))




        recorded_at_estimated = d.pop("recorded_at_estimated")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = MediaTrack.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        error_code = d.pop("error_code", UNSET)

        _discovered_at = d.pop("discovered_at", UNSET)
        discovered_at: datetime.datetime | Unset
        if isinstance(_discovered_at,  Unset):
            discovered_at = UNSET
        else:
            discovered_at = datetime.datetime.fromisoformat(_discovered_at)




        _recorded_at = d.pop("recorded_at", UNSET)
        recorded_at: datetime.datetime | Unset
        if isinstance(_recorded_at,  Unset):
            recorded_at = UNSET
        else:
            recorded_at = datetime.datetime.fromisoformat(_recorded_at)




        duration_seconds = d.pop("duration_seconds", UNSET)

        format_name = d.pop("format_name", UNSET)

        video_codec = d.pop("video_codec", UNSET)

        audio_codec = d.pop("audio_codec", UNSET)

        source_file = cls(
            name=name,
            relative_path=relative_path,
            status=status,
            size_bytes=size_bytes,
            modified_at=modified_at,
            recorded_at_estimated=recorded_at_estimated,
            tracks=tracks,
            error_code=error_code,
            discovered_at=discovered_at,
            recorded_at=recorded_at,
            duration_seconds=duration_seconds,
            format_name=format_name,
            video_codec=video_codec,
            audio_codec=audio_codec,
        )


        source_file.additional_properties = d
        return source_file

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
