from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="AttributionEvent")



@_attrs_define
class AttributionEvent:
    """
        Attributes:
            id (int):
            local_speaker_id (int):
            action (str): validate, correct ; d'autres actions pourront s'ajouter.
            old_person (None | PersonRef):
            old_decision_level (str):
            new_person (None | PersonRef):
            new_decision_level (str):
            author (AttributionAuthor | None):
            created_at (datetime.datetime):
     """

    id: int
    local_speaker_id: int
    action: str
    old_person: None | PersonRef
    old_decision_level: str
    new_person: None | PersonRef
    new_decision_level: str
    author: AttributionAuthor | None
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        id = self.id

        local_speaker_id = self.local_speaker_id

        action = self.action

        old_person: dict[str, Any] | None
        if isinstance(self.old_person, PersonRef):
            old_person = self.old_person.to_dict()
        else:
            old_person = self.old_person

        old_decision_level = self.old_decision_level

        new_person: dict[str, Any] | None
        if isinstance(self.new_person, PersonRef):
            new_person = self.new_person.to_dict()
        else:
            new_person = self.new_person

        new_decision_level = self.new_decision_level

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
            "local_speaker_id": local_speaker_id,
            "action": action,
            "old_person": old_person,
            "old_decision_level": old_decision_level,
            "new_person": new_person,
            "new_decision_level": new_decision_level,
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

        local_speaker_id = d.pop("local_speaker_id")

        action = d.pop("action")

        def _parse_old_person(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                old_person_type_1 = PersonRef.from_dict(data)



                return old_person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        old_person = _parse_old_person(d.pop("old_person"))


        old_decision_level = d.pop("old_decision_level")

        def _parse_new_person(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                new_person_type_1 = PersonRef.from_dict(data)



                return new_person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        new_person = _parse_new_person(d.pop("new_person"))


        new_decision_level = d.pop("new_decision_level")

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




        attribution_event = cls(
            id=id,
            local_speaker_id=local_speaker_id,
            action=action,
            old_person=old_person,
            old_decision_level=old_decision_level,
            new_person=new_person,
            new_decision_level=new_decision_level,
            author=author,
            created_at=created_at,
        )


        attribution_event.additional_properties = d
        return attribution_event

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
