from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="AttributionCandidate")



@_attrs_define
class AttributionCandidate:
    """
        Attributes:
            rank (int):
            score (float): Similarité cosinus entre l'empreinte du Locuteur local et la meilleure référence de la Personne.
            person (PersonRef):
     """

    rank: int
    score: float
    person: PersonRef
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        rank = self.rank

        score = self.score

        person = self.person.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "rank": rank,
            "score": score,
            "person": person,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        rank = d.pop("rank")

        score = d.pop("score")

        person = PersonRef.from_dict(d.pop("person"))




        attribution_candidate = cls(
            rank=rank,
            score=score,
            person=person,
        )


        attribution_candidate.additional_properties = d
        return attribution_candidate

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
