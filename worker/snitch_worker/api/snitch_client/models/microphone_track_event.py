from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.microphone_track_event_action import MicrophoneTrackEventAction
from ..models.microphone_track_event_scope import MicrophoneTrackEventScope
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor
  from ..models.microphone_snapshot import MicrophoneSnapshot





T = TypeVar("T", bound="MicrophoneTrackEvent")



@_attrs_define
class MicrophoneTrackEvent:
    """
        Attributes:
            id (int):
            scope (MicrophoneTrackEventScope):
            source_id (int | None):
            action (MicrophoneTrackEventAction):
            old_rule (MicrophoneSnapshot | None):
            new_rule (MicrophoneSnapshot | None):
            author (AttributionAuthor | None):
            created_at (datetime.datetime):
     """

    id: int
    scope: MicrophoneTrackEventScope
    source_id: int | None
    action: MicrophoneTrackEventAction
    old_rule: MicrophoneSnapshot | None
    new_rule: MicrophoneSnapshot | None
    author: AttributionAuthor | None
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.microphone_snapshot import MicrophoneSnapshot # noqa: PLC0415
        id = self.id

        scope = self.scope.value

        source_id: int | None
        source_id = self.source_id

        action = self.action.value

        old_rule: dict[str, Any] | None
        if isinstance(self.old_rule, MicrophoneSnapshot):
            old_rule = self.old_rule.to_dict()
        else:
            old_rule = self.old_rule

        new_rule: dict[str, Any] | None
        if isinstance(self.new_rule, MicrophoneSnapshot):
            new_rule = self.new_rule.to_dict()
        else:
            new_rule = self.new_rule

        author: dict[str, Any] | None
        if isinstance(self.author, AttributionAuthor):
            author = self.author.to_dict()
        else:
            author = self.author

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "scope": scope,
            "source_id": source_id,
            "action": action,
            "old_rule": old_rule,
            "new_rule": new_rule,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.microphone_snapshot import MicrophoneSnapshot # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        scope = MicrophoneTrackEventScope(d.pop("scope"))




        def _parse_source_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        source_id = _parse_source_id(d.pop("source_id"))


        action = MicrophoneTrackEventAction(d.pop("action"))




        def _parse_old_rule(data: object) -> MicrophoneSnapshot | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                old_rule_type_1 = MicrophoneSnapshot.from_dict(data)



                return old_rule_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MicrophoneSnapshot | None, data)

        old_rule = _parse_old_rule(d.pop("old_rule"))


        def _parse_new_rule(data: object) -> MicrophoneSnapshot | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                new_rule_type_1 = MicrophoneSnapshot.from_dict(data)



                return new_rule_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MicrophoneSnapshot | None, data)

        new_rule = _parse_new_rule(d.pop("new_rule"))


        def _parse_author(data: object) -> AttributionAuthor | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                author_type_1 = AttributionAuthor.from_dict(data)



                return author_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(AttributionAuthor | None, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        microphone_track_event = cls(
            id=id,
            scope=scope,
            source_id=source_id,
            action=action,
            old_rule=old_rule,
            new_rule=new_rule,
            author=author,
            created_at=created_at,
        )


        microphone_track_event.additional_properties = d
        return microphone_track_event

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
