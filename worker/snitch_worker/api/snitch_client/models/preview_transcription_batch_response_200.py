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
            eligible (int):
            already_processed (int):
            already_queued (int):
            unavailable (int):
            selection_required (int):
     """

    eligible: int
    already_processed: int
    already_queued: int
    unavailable: int
    selection_required: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        eligible = self.eligible

        already_processed = self.already_processed

        already_queued = self.already_queued

        unavailable = self.unavailable

        selection_required = self.selection_required


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
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
        eligible = d.pop("eligible")

        already_processed = d.pop("already_processed")

        already_queued = d.pop("already_queued")

        unavailable = d.pop("unavailable")

        selection_required = d.pop("selection_required")

        preview_transcription_batch_response_200 = cls(
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
