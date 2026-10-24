from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_visibility import ClipVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.media_track import MediaTrack
  from ..models.transcription import Transcription





T = TypeVar("T", bound="Clip")



@_attrs_define
class Clip:
    """
        Attributes:
            id (int):
            owner_id (int):
            owner (str):
            visibility (ClipVisibility):
            original_name (str):
            size_bytes (int):
            discovered_at (datetime.datetime):
            recorded_at (datetime.datetime):
            recorded_at_estimated (bool):
            format_name (str):
            video_codec (str):
            audio_codec (str):
            tracks (list[MediaTrack]):
            uploaders (list[str]):
            duration_seconds (float | None | Unset):
            status (str | Unset):
            sources (list[str] | Unset):
            transcription (Transcription | Unset):
     """

    id: int
    owner_id: int
    owner: str
    visibility: ClipVisibility
    original_name: str
    size_bytes: int
    discovered_at: datetime.datetime
    recorded_at: datetime.datetime
    recorded_at_estimated: bool
    format_name: str
    video_codec: str
    audio_codec: str
    tracks: list[MediaTrack]
    uploaders: list[str]
    duration_seconds: float | None | Unset = UNSET
    status: str | Unset = UNSET
    sources: list[str] | Unset = UNSET
    transcription: Transcription | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.media_track import MediaTrack # noqa: PLC0415
        from ..models.transcription import Transcription # noqa: PLC0415
        id = self.id

        owner_id = self.owner_id

        owner = self.owner

        visibility = self.visibility.value

        original_name = self.original_name

        size_bytes = self.size_bytes

        discovered_at = self.discovered_at.isoformat()

        recorded_at = self.recorded_at.isoformat()

        recorded_at_estimated = self.recorded_at_estimated

        format_name = self.format_name

        video_codec = self.video_codec

        audio_codec = self.audio_codec

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        uploaders = self.uploaders



        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds

        status = self.status

        sources: list[str] | Unset = UNSET
        if not isinstance(self.sources, Unset):
            sources = self.sources



        transcription: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transcription, Unset):
            transcription = self.transcription.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "owner_id": owner_id,
            "owner": owner,
            "visibility": visibility,
            "original_name": original_name,
            "size_bytes": size_bytes,
            "discovered_at": discovered_at,
            "recorded_at": recorded_at,
            "recorded_at_estimated": recorded_at_estimated,
            "format_name": format_name,
            "video_codec": video_codec,
            "audio_codec": audio_codec,
            "tracks": tracks,
            "uploaders": uploaders,
        })
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds
        if status is not UNSET:
            field_dict["status"] = status
        if sources is not UNSET:
            field_dict["sources"] = sources
        if transcription is not UNSET:
            field_dict["transcription"] = transcription

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.media_track import MediaTrack # noqa: PLC0415
        from ..models.transcription import Transcription # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        owner_id = d.pop("owner_id")

        owner = d.pop("owner")

        visibility = ClipVisibility(d.pop("visibility"))




        original_name = d.pop("original_name")

        size_bytes = d.pop("size_bytes")

        discovered_at = datetime.datetime.fromisoformat(d.pop("discovered_at"))




        recorded_at = datetime.datetime.fromisoformat(d.pop("recorded_at"))




        recorded_at_estimated = d.pop("recorded_at_estimated")

        format_name = d.pop("format_name")

        video_codec = d.pop("video_codec")

        audio_codec = d.pop("audio_codec")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = MediaTrack.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        uploaders = cast(list[str], d.pop("uploaders"))


        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        status = d.pop("status", UNSET)

        sources = cast(list[str], d.pop("sources", UNSET))


        _transcription = d.pop("transcription", UNSET)
        transcription: Transcription | Unset
        if isinstance(_transcription,  Unset):
            transcription = UNSET
        else:
            transcription = Transcription.from_dict(_transcription)




        clip = cls(
            id=id,
            owner_id=owner_id,
            owner=owner,
            visibility=visibility,
            original_name=original_name,
            size_bytes=size_bytes,
            discovered_at=discovered_at,
            recorded_at=recorded_at,
            recorded_at_estimated=recorded_at_estimated,
            format_name=format_name,
            video_codec=video_codec,
            audio_codec=audio_codec,
            tracks=tracks,
            uploaders=uploaders,
            duration_seconds=duration_seconds,
            status=status,
            sources=sources,
            transcription=transcription,
        )


        clip.additional_properties = d
        return clip

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
