from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_trace_segments_item_candidates_item import VoiceTraceSegmentsItemCandidatesItem





T = TypeVar("T", bound="VoiceTraceSegmentsItem")



@_attrs_define
class VoiceTraceSegmentsItem:
    """
        Attributes:
            index (int):
            start (float):
            end (float):
            candidates (list[VoiceTraceSegmentsItemCandidatesItem]):
            deduced_local_speaker_id (int | None):
            local_speaker_id (int | None):
            corrected (bool):
     """

    index: int
    start: float
    end: float
    candidates: list[VoiceTraceSegmentsItemCandidatesItem]
    deduced_local_speaker_id: int | None
    local_speaker_id: int | None
    corrected: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_trace_segments_item_candidates_item import VoiceTraceSegmentsItemCandidatesItem # noqa: PLC0415
        index = self.index

        start = self.start

        end = self.end

        candidates = []
        for candidates_item_data in self.candidates:
            candidates_item = candidates_item_data.to_dict()
            candidates.append(candidates_item)



        deduced_local_speaker_id: int | None
        deduced_local_speaker_id = self.deduced_local_speaker_id

        local_speaker_id: int | None
        local_speaker_id = self.local_speaker_id

        corrected = self.corrected


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "index": index,
            "start": start,
            "end": end,
            "candidates": candidates,
            "deduced_local_speaker_id": deduced_local_speaker_id,
            "local_speaker_id": local_speaker_id,
            "corrected": corrected,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_trace_segments_item_candidates_item import VoiceTraceSegmentsItemCandidatesItem # noqa: PLC0415
        d = dict(src_dict)
        index = d.pop("index")

        start = d.pop("start")

        end = d.pop("end")

        candidates = []
        _candidates = d.pop("candidates")
        for candidates_item_data in (_candidates):
            candidates_item = VoiceTraceSegmentsItemCandidatesItem.from_dict(candidates_item_data)



            candidates.append(candidates_item)


        def _parse_deduced_local_speaker_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        deduced_local_speaker_id = _parse_deduced_local_speaker_id(d.pop("deduced_local_speaker_id"))


        def _parse_local_speaker_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        local_speaker_id = _parse_local_speaker_id(d.pop("local_speaker_id"))


        corrected = d.pop("corrected")

        voice_trace_segments_item = cls(
            index=index,
            start=start,
            end=end,
            candidates=candidates,
            deduced_local_speaker_id=deduced_local_speaker_id,
            local_speaker_id=local_speaker_id,
            corrected=corrected,
        )


        voice_trace_segments_item.additional_properties = d
        return voice_trace_segments_item

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
