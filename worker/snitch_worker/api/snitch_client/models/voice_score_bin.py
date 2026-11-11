from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceScoreBin")



@_attrs_define
class VoiceScoreBin:
    """ Classe de score de largeur 0,05 ; un score négatif est compté dans la première classe, un score de 1 dans la
    dernière. Un score est une similarité cosinus, jamais une probabilité.

        Attributes:
            from_ (float):
            to (float):
            same_person (int): Scores d'un test contre la Référence de sa propre Personne.
            other_person (int): Scores d'un test contre la Référence d'une autre Personne.
     """

    from_: float
    to: float
    same_person: int
    other_person: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_

        to = self.to

        same_person = self.same_person

        other_person = self.other_person


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "from": from_,
            "to": to,
            "same_person": same_person,
            "other_person": other_person,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = d.pop("from")

        to = d.pop("to")

        same_person = d.pop("same_person")

        other_person = d.pop("other_person")

        voice_score_bin = cls(
            from_=from_,
            to=to,
            same_person=same_person,
            other_person=other_person,
        )


        voice_score_bin.additional_properties = d
        return voice_score_bin

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
