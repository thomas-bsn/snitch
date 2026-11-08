from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_audio_dedup_segment import WorkerAudioDedupSegment





T = TypeVar("T", bound="WorkerAudioDedupTrack")



@_attrs_define
class WorkerAudioDedupTrack:
    """
        Attributes:
            track_index (int):
            job_id (int):
            start_seconds (float):
            segments (list[WorkerAudioDedupSegment]):
     """

    track_index: int
    job_id: int
    start_seconds: float
    segments: list[WorkerAudioDedupSegment]





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_audio_dedup_segment import WorkerAudioDedupSegment # noqa: PLC0415
        track_index = self.track_index

        job_id = self.job_id

        start_seconds = self.start_seconds

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "track_index": track_index,
            "job_id": job_id,
            "start_seconds": start_seconds,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_audio_dedup_segment import WorkerAudioDedupSegment # noqa: PLC0415
        d = dict(src_dict)
        track_index = d.pop("track_index")

        job_id = d.pop("job_id")

        start_seconds = d.pop("start_seconds")

        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = WorkerAudioDedupSegment.from_dict(segments_item_data)



            segments.append(segments_item)


        worker_audio_dedup_track = cls(
            track_index=track_index,
            job_id=job_id,
            start_seconds=start_seconds,
            segments=segments,
        )

        return worker_audio_dedup_track
