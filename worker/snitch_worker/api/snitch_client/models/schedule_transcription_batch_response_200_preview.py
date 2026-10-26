from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ScheduleTranscriptionBatchResponse200Preview")



@_attrs_define
class ScheduleTranscriptionBatchResponse200Preview:
    """
        Attributes:
            corrected (int | Unset):
            corrected_segments (int | Unset):
            eligible (int | Unset):
            already_processed (int | Unset):
            already_queued (int | Unset):
            unavailable (int | Unset):
            selection_required (int | Unset):
     """

    corrected: int | Unset = UNSET
    corrected_segments: int | Unset = UNSET
    eligible: int | Unset = UNSET
    already_processed: int | Unset = UNSET
    already_queued: int | Unset = UNSET
    unavailable: int | Unset = UNSET
    selection_required: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        corrected = self.corrected

        corrected_segments = self.corrected_segments

        eligible = self.eligible

        already_processed = self.already_processed

        already_queued = self.already_queued

        unavailable = self.unavailable

        selection_required = self.selection_required


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if corrected is not UNSET:
            field_dict["corrected"] = corrected
        if corrected_segments is not UNSET:
            field_dict["corrected_segments"] = corrected_segments
        if eligible is not UNSET:
            field_dict["eligible"] = eligible
        if already_processed is not UNSET:
            field_dict["already_processed"] = already_processed
        if already_queued is not UNSET:
            field_dict["already_queued"] = already_queued
        if unavailable is not UNSET:
            field_dict["unavailable"] = unavailable
        if selection_required is not UNSET:
            field_dict["selection_required"] = selection_required

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        corrected = d.pop("corrected", UNSET)

        corrected_segments = d.pop("corrected_segments", UNSET)

        eligible = d.pop("eligible", UNSET)

        already_processed = d.pop("already_processed", UNSET)

        already_queued = d.pop("already_queued", UNSET)

        unavailable = d.pop("unavailable", UNSET)

        selection_required = d.pop("selection_required", UNSET)

        schedule_transcription_batch_response_200_preview = cls(
            corrected=corrected,
            corrected_segments=corrected_segments,
            eligible=eligible,
            already_processed=already_processed,
            already_queued=already_queued,
            unavailable=unavailable,
            selection_required=selection_required,
        )


        schedule_transcription_batch_response_200_preview.additional_properties = d
        return schedule_transcription_batch_response_200_preview

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
