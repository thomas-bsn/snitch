from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.attribution_validation_state import AttributionValidationState
from ..models.decision_level import DecisionLevel
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.attribution_author import AttributionAuthor
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="Attribution")



@_attrs_define
class Attribution:
    """
        Attributes:
            id (int):
            local_speaker_id (int):
            person (None | PersonRef):
            decision_level (DecisionLevel): human : validation humaine ; automatic, suggestion, ambiguous : issus du
                rapprochement (tickets suivants) ; unknown : aucune attribution.
            validation_state (AttributionValidationState): validated : validation humaine (fait foi, source d'Échantillon
                vocal validé).
            confidence (float | None): Confiance automatique du rapprochement, absente d'une validation humaine.
            origin (str): human, puis matching ou microphone_track selon les tickets suivants.
            author (AttributionAuthor | None):
            revision (int):
            decided_at (datetime.datetime):
     """

    id: int
    local_speaker_id: int
    person: None | PersonRef
    decision_level: DecisionLevel
    validation_state: AttributionValidationState
    confidence: float | None
    origin: str
    author: AttributionAuthor | None
    revision: int
    decided_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        id = self.id

        local_speaker_id = self.local_speaker_id

        person: dict[str, Any] | None
        if isinstance(self.person, PersonRef):
            person = self.person.to_dict()
        else:
            person = self.person

        decision_level = self.decision_level.value

        validation_state = self.validation_state.value

        confidence: float | None
        confidence = self.confidence

        origin = self.origin

        author: dict[str, Any] | None
        if isinstance(self.author, AttributionAuthor):
            author = self.author.to_dict()
        else:
            author = self.author

        revision = self.revision

        decided_at = self.decided_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "local_speaker_id": local_speaker_id,
            "person": person,
            "decision_level": decision_level,
            "validation_state": validation_state,
            "confidence": confidence,
            "origin": origin,
            "author": author,
            "revision": revision,
            "decided_at": decided_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attribution_author import AttributionAuthor # noqa: PLC0415
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        local_speaker_id = d.pop("local_speaker_id")

        def _parse_person(data: object) -> None | PersonRef:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                person_type_1 = PersonRef.from_dict(data)



                return person_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | PersonRef, data)

        person = _parse_person(d.pop("person"))


        decision_level = DecisionLevel(d.pop("decision_level"))




        validation_state = AttributionValidationState(d.pop("validation_state"))




        def _parse_confidence(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        confidence = _parse_confidence(d.pop("confidence"))


        origin = d.pop("origin")

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


        revision = d.pop("revision")

        decided_at = datetime.datetime.fromisoformat(d.pop("decided_at"))




        attribution = cls(
            id=id,
            local_speaker_id=local_speaker_id,
            person=person,
            decision_level=decision_level,
            validation_state=validation_state,
            confidence=confidence,
            origin=origin,
            author=author,
            revision=revision,
            decided_at=decided_at,
        )


        attribution.additional_properties = d
        return attribution

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
