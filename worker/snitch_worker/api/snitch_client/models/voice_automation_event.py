from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_automation_event_action import VoiceAutomationEventAction
from typing import cast
import datetime






T = TypeVar("T", bound="VoiceAutomationEvent")



@_attrs_define
class VoiceAutomationEvent:
    """
        Attributes:
            action (VoiceAutomationEventAction):
            author (str):
            criteria_met (bool):
            forced (bool): Activation confirmée explicitement malgré des critères non atteints.
            measured_precision (float | None):
            confirmations (int):
            precision_threshold (float):
            min_confirmations (int):
            created_at (datetime.datetime):
     """

    action: VoiceAutomationEventAction
    author: str
    criteria_met: bool
    forced: bool
    measured_precision: float | None
    confirmations: int
    precision_threshold: float
    min_confirmations: int
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        author = self.author

        criteria_met = self.criteria_met

        forced = self.forced

        measured_precision: float | None
        measured_precision = self.measured_precision

        confirmations = self.confirmations

        precision_threshold = self.precision_threshold

        min_confirmations = self.min_confirmations

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "action": action,
            "author": author,
            "criteria_met": criteria_met,
            "forced": forced,
            "measured_precision": measured_precision,
            "confirmations": confirmations,
            "precision_threshold": precision_threshold,
            "min_confirmations": min_confirmations,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = VoiceAutomationEventAction(d.pop("action"))




        author = d.pop("author")

        criteria_met = d.pop("criteria_met")

        forced = d.pop("forced")

        def _parse_measured_precision(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        measured_precision = _parse_measured_precision(d.pop("measured_precision"))


        confirmations = d.pop("confirmations")

        precision_threshold = d.pop("precision_threshold")

        min_confirmations = d.pop("min_confirmations")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        voice_automation_event = cls(
            action=action,
            author=author,
            criteria_met=criteria_met,
            forced=forced,
            measured_precision=measured_precision,
            confirmations=confirmations,
            precision_threshold=precision_threshold,
            min_confirmations=min_confirmations,
            created_at=created_at,
        )


        voice_automation_event.additional_properties = d
        return voice_automation_event

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
