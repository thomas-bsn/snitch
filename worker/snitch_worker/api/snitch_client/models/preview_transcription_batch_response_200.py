from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PreviewTranscriptionBatchResponse200")



@_attrs_define
class PreviewTranscriptionBatchResponse200:
    """
        Attributes:
            corrected (int): Clips autrement éligibles qui portent des corrections ; exclus de eligible sauf demande
                explicite.
            corrected_segments (int): Corrections actives portées par ces clips.
            eligible (int):
            already_processed (int):
            already_queued (int):
            unavailable (int):
            selection_required (int):
     """

    corrected: int
    corrected_segments: int
    eligible: int
    already_processed: int
    already_queued: int
    unavailable: int
    selection_required: int
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
            "corrected": corrected,
            "corrected_segments": corrected_segments,
            "eligible": eligible,
            "already_processed": already_processed,
            "already_queued": already_queued,
            "unavailable": unavailable,
            "selection_required": selection_required,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        corrected = d.pop("corrected")

        corrected_segments = d.pop("corrected_segments")

        eligible = d.pop("eligible")

        already_processed = d.pop("already_processed")

        already_queued = d.pop("already_queued")

        unavailable = d.pop("unavailable")

        selection_required = d.pop("selection_required")

        preview_transcription_batch_response_200 = cls(
            corrected=corrected,
            corrected_segments=corrected_segments,
            eligible=eligible,
            already_processed=already_processed,
            already_queued=already_queued,
            unavailable=unavailable,
            selection_required=selection_required,
        )


        preview_transcription_batch_response_200.additional_properties = d
        return preview_transcription_batch_response_200

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
