from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor





T = TypeVar("T", bound="SegmentCorrectionExportItem")



@_attrs_define
class SegmentCorrectionExportItem:
    """
        Attributes:
            clip_id (int):
            transcription_job_id (int):
            audio_stream_index (int): Piste audio transcrite (-1 si absente du résultat).
            index (int): Rang du segment dans la transcription.
            start (float):
            end (float):
            original_text (str): Texte du worker, jamais modifié.
            corrected_text (str): Texte corrigé courant.
            author (AttributionAuthor):
            corrected_at (datetime.datetime):
            version (int):
     """

    clip_id: int
    transcription_job_id: int
    audio_stream_index: int
    index: int
    start: float
    end: float
    original_text: str
    corrected_text: str
    author: AttributionAuthor
    corrected_at: datetime.datetime
    version: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        clip_id = self.clip_id

        transcription_job_id = self.transcription_job_id

        audio_stream_index = self.audio_stream_index

        index = self.index

        start = self.start

        end = self.end

        original_text = self.original_text

        corrected_text = self.corrected_text

        author = self.author.to_dict()

        corrected_at = self.corrected_at.isoformat()

        version = self.version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "transcription_job_id": transcription_job_id,
            "audio_stream_index": audio_stream_index,
            "index": index,
            "start": start,
            "end": end,
            "original_text": original_text,
            "corrected_text": corrected_text,
            "author": author,
            "corrected_at": corrected_at,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        transcription_job_id = d.pop("transcription_job_id")

        audio_stream_index = d.pop("audio_stream_index")

        index = d.pop("index")

        start = d.pop("start")

        end = d.pop("end")

        original_text = d.pop("original_text")

        corrected_text = d.pop("corrected_text")

        author = AttributionAuthor.from_dict(d.pop("author"))




        corrected_at = datetime.datetime.fromisoformat(d.pop("corrected_at"))




        version = d.pop("version")

        segment_correction_export_item = cls(
            clip_id=clip_id,
            transcription_job_id=transcription_job_id,
            audio_stream_index=audio_stream_index,
            index=index,
            start=start,
            end=end,
            original_text=original_text,
            corrected_text=corrected_text,
            author=author,
            corrected_at=corrected_at,
            version=version,
        )


        segment_correction_export_item.additional_properties = d
        return segment_correction_export_item

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
