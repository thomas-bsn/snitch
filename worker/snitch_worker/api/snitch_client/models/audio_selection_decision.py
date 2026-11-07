from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_selection_decision_action import AudioSelectionDecisionAction






T = TypeVar("T", bound="AudioSelectionDecision")



@_attrs_define
class AudioSelectionDecision:
    """
        Attributes:
            track_index (int):
            action (AudioSelectionDecisionAction):
            forced (bool):
            reason (str):
     """

    track_index: int
    action: AudioSelectionDecisionAction
    forced: bool
    reason: str





    def to_dict(self) -> dict[str, Any]:
        track_index = self.track_index

        action = self.action.value

        forced = self.forced

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "track_index": track_index,
            "action": action,
            "forced": forced,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        track_index = d.pop("track_index")

        action = AudioSelectionDecisionAction(d.pop("action"))




        forced = d.pop("forced")

        reason = d.pop("reason")

        audio_selection_decision = cls(
            track_index=track_index,
            action=action,
            forced=forced,
            reason=reason,
        )

        return audio_selection_decision
