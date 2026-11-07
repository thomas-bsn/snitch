from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerAudioDedupSegment")



@_attrs_define
class WorkerAudioDedupSegment:
    """
        Attributes:
            index (int):
            start (float):
            end (float):
     """

    index: int
    start: float
    end: float





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        start = self.start

        end = self.end


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "start": start,
            "end": end,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        start = d.pop("start")

        end = d.pop("end")

        worker_audio_dedup_segment = cls(
            index=index,
            start=start,
            end=end,
        )

        return worker_audio_dedup_segment
