from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcript_revision_time_base import TranscriptRevisionTimeBase
from typing import cast

if TYPE_CHECKING:
  from ..models.transcript_manifest_segment import TranscriptManifestSegment





T = TypeVar("T", bound="TranscriptRevision")



@_attrs_define
class TranscriptRevision:
    """
        Attributes:
            id (str):
            clip_id (int):
            plan_id (int | None):
            legacy_job_id (int | None):
            empty (bool):
            time_base (TranscriptRevisionTimeBase):
            segments (list[TranscriptManifestSegment]):
     """

    id: str
    clip_id: int
    plan_id: int | None
    legacy_job_id: int | None
    empty: bool
    time_base: TranscriptRevisionTimeBase
    segments: list[TranscriptManifestSegment]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcript_manifest_segment import TranscriptManifestSegment # noqa: PLC0415
        id = self.id

        clip_id = self.clip_id

        plan_id: int | None
        plan_id = self.plan_id

        legacy_job_id: int | None
        legacy_job_id = self.legacy_job_id

        empty = self.empty

        time_base = self.time_base.value

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "clip_id": clip_id,
            "plan_id": plan_id,
            "legacy_job_id": legacy_job_id,
            "empty": empty,
            "time_base": time_base,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transcript_manifest_segment import TranscriptManifestSegment # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        clip_id = d.pop("clip_id")

        def _parse_plan_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        plan_id = _parse_plan_id(d.pop("plan_id"))


        def _parse_legacy_job_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        legacy_job_id = _parse_legacy_job_id(d.pop("legacy_job_id"))


        empty = d.pop("empty")

        time_base = TranscriptRevisionTimeBase(d.pop("time_base"))




        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = TranscriptManifestSegment.from_dict(segments_item_data)



            segments.append(segments_item)


        transcript_revision = cls(
            id=id,
            clip_id=clip_id,
            plan_id=plan_id,
            legacy_job_id=legacy_job_id,
            empty=empty,
            time_base=time_base,
            segments=segments,
        )

        return transcript_revision
