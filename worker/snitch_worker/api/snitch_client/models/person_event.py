from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.person_event_action import PersonEventAction
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="PersonEvent")



@_attrs_define
class PersonEvent:
    """
        Attributes:
            id (int):
            action (PersonEventAction):
            person (PersonRef):
            target (None | PersonRef):
            attribution_count (int):
            sample_count (int):
            microphone_rule_count (int):
            author (AttributionAuthor | None):
            created_at (datetime.datetime):
     """

    id: int
    action: PersonEventAction
    person: PersonRef
    target: None | PersonRef
    attribution_count: int
    sample_count: int
    microphone_rule_count: int
    author: AttributionAuthor | None
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        id = self.id

        action = self.action.value

        person = self.person.to_dict()

        target: dict[str, Any] | None
        if isinstance(self.target, PersonRef):
            target = self.target.to_dict()
        else:
            target = self.target

        attribution_count = self.attribution_count

        sample_count = self.sample_count

        microphone_rule_count = self.microphone_rule_count

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
            "action": action,
            "person": person,
            "target": target,
            "attribution_count": attribution_count,
            "sample_count": sample_count,
            "microphone_rule_count": microphone_rule_count,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        action = PersonEventAction(d.pop("action"))




        person = PersonRef.from_dict(d.pop("person"))




        def _parse_target(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                target_type_1 = PersonRef.from_dict(data)



                return target_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        target = _parse_target(d.pop("target"))


        attribution_count = d.pop("attribution_count")

        sample_count = d.pop("sample_count")

        microphone_rule_count = d.pop("microphone_rule_count")

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




        person_event = cls(
            id=id,
            action=action,
            person=person,
            target=target,
            attribution_count=attribution_count,
            sample_count=sample_count,
            microphone_rule_count=microphone_rule_count,
            author=author,
            created_at=created_at,
        )


        person_event.additional_properties = d
        return person_event

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
