from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.microphone_detected_track import MicrophoneDetectedTrack
  from ..models.microphone_source_rule_status_counts import MicrophoneSourceRuleStatusCounts
  from ..models.microphone_source_rule_value import MicrophoneSourceRuleValue





T = TypeVar("T", bound="MicrophoneSourceRule")



@_attrs_define
class MicrophoneSourceRule:
    """
        Attributes:
            source_id (int):
            rule (MicrophoneSourceRuleValue | None):
            detected_tracks (list[MicrophoneDetectedTrack]):
            status_counts (MicrophoneSourceRuleStatusCounts): Après une écriture : nombre de clips par état de désignation
                (designated, track_not_found, ambiguous_track, conflicting_sources, blocked_by_validation, disabled, none). Vide
                en lecture.
     """

    source_id: int
    rule: MicrophoneSourceRuleValue | None
    detected_tracks: list[MicrophoneDetectedTrack]
    status_counts: MicrophoneSourceRuleStatusCounts
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.microphone_detected_track import MicrophoneDetectedTrack # noqa: PLC0415
        from ..models.microphone_source_rule_status_counts import MicrophoneSourceRuleStatusCounts # noqa: PLC0415
        from ..models.microphone_source_rule_value import MicrophoneSourceRuleValue # noqa: PLC0415
        source_id = self.source_id

        rule: dict[str, Any] | None
        if isinstance(self.rule, MicrophoneSourceRuleValue):
            rule = self.rule.to_dict()
        else:
            rule = self.rule

        detected_tracks = []
        for detected_tracks_item_data in self.detected_tracks:
            detected_tracks_item = detected_tracks_item_data.to_dict()
            detected_tracks.append(detected_tracks_item)



        status_counts = self.status_counts.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "source_id": source_id,
            "rule": rule,
            "detected_tracks": detected_tracks,
            "status_counts": status_counts,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.microphone_detected_track import MicrophoneDetectedTrack # noqa: PLC0415
        from ..models.microphone_source_rule_status_counts import MicrophoneSourceRuleStatusCounts # noqa: PLC0415
        from ..models.microphone_source_rule_value import MicrophoneSourceRuleValue # noqa: PLC0415
        d = dict(src_dict)
        source_id = d.pop("source_id")

        def _parse_rule(data: object) -> MicrophoneSourceRuleValue | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                rule_type_1 = MicrophoneSourceRuleValue.from_dict(data)



                return rule_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MicrophoneSourceRuleValue | None, data)

        rule = _parse_rule(d.pop("rule"))


        detected_tracks = []
        _detected_tracks = d.pop("detected_tracks")
        for detected_tracks_item_data in (_detected_tracks):
            detected_tracks_item = MicrophoneDetectedTrack.from_dict(detected_tracks_item_data)



            detected_tracks.append(detected_tracks_item)


        status_counts = MicrophoneSourceRuleStatusCounts.from_dict(d.pop("status_counts"))




        microphone_source_rule = cls(
            source_id=source_id,
            rule=rule,
            detected_tracks=detected_tracks,
            status_counts=status_counts,
        )


        microphone_source_rule.additional_properties = d
        return microphone_source_rule

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
