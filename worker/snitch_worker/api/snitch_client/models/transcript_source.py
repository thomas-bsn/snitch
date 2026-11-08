from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TranscriptSource")



@_attrs_define
class TranscriptSource:
    """
        Attributes:
            track_index (int): Indice ffprobe réel ; -1 réservé à une ancienne transcription sans indice enregistré.
            job_id (int):
            segment_index (int):
            word_index (int | Unset):
     """

    track_index: int
    job_id: int
    segment_index: int
    word_index: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        track_index = self.track_index

        job_id = self.job_id

        segment_index = self.segment_index

        word_index = self.word_index


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "track_index": track_index,
            "job_id": job_id,
            "segment_index": segment_index,
        })
        if word_index is not UNSET:
            field_dict["word_index"] = word_index

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        track_index = d.pop("track_index")

        job_id = d.pop("job_id")

        segment_index = d.pop("segment_index")

        word_index = d.pop("word_index", UNSET)

        transcript_source = cls(
            track_index=track_index,
            job_id=job_id,
            segment_index=segment_index,
            word_index=word_index,
        )

        return transcript_source
