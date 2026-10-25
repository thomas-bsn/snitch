from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.decision_level import DecisionLevel






T = TypeVar("T", bound="ClipPerson")



@_attrs_define
class ClipPerson:
    """
        Attributes:
            id (int):
            display_name (str):
            decision_level (DecisionLevel): human : validation humaine ; automatic, suggestion, ambiguous : issus du
                rapprochement en base (automatic seulement lorsque l'administrateur a activé l'automatisation) ; unknown :
                aucune attribution ou aucune Personne assez proche.
     """

    id: int
    display_name: str
    decision_level: DecisionLevel
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        display_name = self.display_name

        decision_level = self.decision_level.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "display_name": display_name,
            "decision_level": decision_level,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        display_name = d.pop("display_name")

        decision_level = DecisionLevel(d.pop("decision_level"))




        clip_person = cls(
            id=id,
            display_name=display_name,
            decision_level=decision_level,
        )


        clip_person.additional_properties = d
        return clip_person

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
