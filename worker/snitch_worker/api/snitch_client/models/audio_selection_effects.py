from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audio_loss_preview import AudioLossPreview





T = TypeVar("T", bound="AudioSelectionEffects")



@_attrs_define
class AudioSelectionEffects:
    """
        Attributes:
            added_steps (list[str]):
            invalidated_steps (list[str]):
            track_indices (list[int]):
            losses (AudioLossPreview):
            worker_compatible (bool):
            stake_version (str):
     """

    added_steps: list[str]
    invalidated_steps: list[str]
    track_indices: list[int]
    losses: AudioLossPreview
    worker_compatible: bool
    stake_version: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        added_steps = self.added_steps



        invalidated_steps = self.invalidated_steps



        track_indices = self.track_indices



        losses = self.losses.to_dict()

        worker_compatible = self.worker_compatible

        stake_version = self.stake_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added_steps": added_steps,
            "invalidated_steps": invalidated_steps,
            "track_indices": track_indices,
            "losses": losses,
            "worker_compatible": worker_compatible,
            "stake_version": stake_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        d = dict(src_dict)
        added_steps = cast(list[str], d.pop("added_steps"))


        invalidated_steps = cast(list[str], d.pop("invalidated_steps"))


        track_indices = cast(list[int], d.pop("track_indices"))


        losses = AudioLossPreview.from_dict(d.pop("losses"))




        worker_compatible = d.pop("worker_compatible")

        stake_version = d.pop("stake_version")

        audio_selection_effects = cls(
            added_steps=added_steps,
            invalidated_steps=invalidated_steps,
            track_indices=track_indices,
            losses=losses,
            worker_compatible=worker_compatible,
            stake_version=stake_version,
        )

        return audio_selection_effects
