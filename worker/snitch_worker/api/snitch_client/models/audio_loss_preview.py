from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="AudioLossPreview")



@_attrs_define
class AudioLossPreview:
    """
        Attributes:
            expected_revision_id (str):
            previewed_at (datetime.datetime):
            correction_count (int):
            validated_voice_count (int):
            replaced_track_indices (list[int]):
            invalidated_steps (list[str]):
            confirmation_required (bool):
     """

    expected_revision_id: str
    previewed_at: datetime.datetime
    correction_count: int
    validated_voice_count: int
    replaced_track_indices: list[int]
    invalidated_steps: list[str]
    confirmation_required: bool





    def to_dict(self) -> dict[str, Any]:
        expected_revision_id = self.expected_revision_id

        previewed_at = self.previewed_at.isoformat()

        correction_count = self.correction_count

        validated_voice_count = self.validated_voice_count

        replaced_track_indices = self.replaced_track_indices



        invalidated_steps = self.invalidated_steps



        confirmation_required = self.confirmation_required


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "expected_revision_id": expected_revision_id,
            "previewed_at": previewed_at,
            "correction_count": correction_count,
            "validated_voice_count": validated_voice_count,
            "replaced_track_indices": replaced_track_indices,
            "invalidated_steps": invalidated_steps,
            "confirmation_required": confirmation_required,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        expected_revision_id = d.pop("expected_revision_id")

        previewed_at = datetime.datetime.fromisoformat(d.pop("previewed_at"))




        correction_count = d.pop("correction_count")

        validated_voice_count = d.pop("validated_voice_count")

        replaced_track_indices = cast(list[int], d.pop("replaced_track_indices"))


        invalidated_steps = cast(list[str], d.pop("invalidated_steps"))


        confirmation_required = d.pop("confirmation_required")

        audio_loss_preview = cls(
            expected_revision_id=expected_revision_id,
            previewed_at=previewed_at,
            correction_count=correction_count,
            validated_voice_count=validated_voice_count,
            replaced_track_indices=replaced_track_indices,
            invalidated_steps=invalidated_steps,
            confirmation_required=confirmation_required,
        )

        return audio_loss_preview
