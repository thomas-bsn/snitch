from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_track_decision_action import AudioTrackDecisionAction
from typing import cast
import datetime






T = TypeVar("T", bound="AudioTrackDecision")



@_attrs_define
class AudioTrackDecision:
    """
        Attributes:
            track_index (int):
            action (AudioTrackDecisionAction):
            forced (bool):
            reason (str):
            actor_id (int):
            decided_at (datetime.datetime):
     """

    track_index: int
    action: AudioTrackDecisionAction
    forced: bool
    reason: str
    actor_id: int
    decided_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        track_index = self.track_index

        action = self.action.value

        forced = self.forced

        reason = self.reason

        actor_id = self.actor_id

        decided_at = self.decided_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "track_index": track_index,
            "action": action,
            "forced": forced,
            "reason": reason,
            "actor_id": actor_id,
            "decided_at": decided_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        track_index = d.pop("track_index")

        action = AudioTrackDecisionAction(d.pop("action"))




        forced = d.pop("forced")

        reason = d.pop("reason")

        actor_id = d.pop("actor_id")

        decided_at = datetime.datetime.fromisoformat(d.pop("decided_at"))




        audio_track_decision = cls(
            track_index=track_index,
            action=action,
            forced=forced,
            reason=reason,
            actor_id=actor_id,
            decided_at=decided_at,
        )

        return audio_track_decision
