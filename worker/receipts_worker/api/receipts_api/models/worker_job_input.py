from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.job import Job
  from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem





T = TypeVar("T", bound="WorkerJobInput")



@_attrs_define
class WorkerJobInput:
    """
        Attributes:
            job (Job):
            original_name (str):
            size_bytes (int):
            video_codec (str):
            audio_codec (str):
            tracks (list[WorkerJobInputTracksItem]):
            media_url (str):
            duration_seconds (float | None | Unset):
     """

    job: Job
    original_name: str
    size_bytes: int
    video_codec: str
    audio_codec: str
    tracks: list[WorkerJobInputTracksItem]
    media_url: str
    duration_seconds: float | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job import Job # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        job = self.job.to_dict()

        original_name = self.original_name

        size_bytes = self.size_bytes

        video_codec = self.video_codec

        audio_codec = self.audio_codec

        tracks = []
        for tracks_item_data in self.tracks:
            tracks_item = tracks_item_data.to_dict()
            tracks.append(tracks_item)



        media_url = self.media_url

        duration_seconds: float | None | Unset
        if isinstance(self.duration_seconds, Unset):
            duration_seconds = UNSET
        else:
            duration_seconds = self.duration_seconds


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job": job,
            "original_name": original_name,
            "size_bytes": size_bytes,
            "video_codec": video_codec,
            "audio_codec": audio_codec,
            "tracks": tracks,
            "media_url": media_url,
        })
        if duration_seconds is not UNSET:
            field_dict["duration_seconds"] = duration_seconds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job import Job # noqa: PLC0415
        from ..models.worker_job_input_tracks_item import WorkerJobInputTracksItem # noqa: PLC0415
        d = dict(src_dict)
        job = Job.from_dict(d.pop("job"))




        original_name = d.pop("original_name")

        size_bytes = d.pop("size_bytes")

        video_codec = d.pop("video_codec")

        audio_codec = d.pop("audio_codec")

        tracks = []
        _tracks = d.pop("tracks")
        for tracks_item_data in (_tracks):
            tracks_item = WorkerJobInputTracksItem.from_dict(tracks_item_data)



            tracks.append(tracks_item)


        media_url = d.pop("media_url")

        def _parse_duration_seconds(data: object) -> float | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(float | None | Unset, data)

        duration_seconds = _parse_duration_seconds(d.pop("duration_seconds", UNSET))


        worker_job_input = cls(
            job=job,
            original_name=original_name,
            size_bytes=size_bytes,
            video_codec=video_codec,
            audio_codec=audio_codec,
            tracks=tracks,
            media_url=media_url,
            duration_seconds=duration_seconds,
        )


        worker_job_input.additional_properties = d
        return worker_job_input

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
