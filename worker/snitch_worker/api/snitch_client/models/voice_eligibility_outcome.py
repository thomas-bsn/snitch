from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceEligibilityOutcome")



@_attrs_define
class VoiceEligibilityOutcome:
    """ Empreintes de Locuteur local de production par éligibilité, et Personnes qui ont une Référence.

        Attributes:
            present (int): Empreintes de Locuteur local de production présentes.
            comparable (int): Comparables.
            contributive (int): Contributives (comparables comprises).
            too_short (int): Ni comparables ni contributives : Parole nette trop courte.
            overlap_only (int): Rejetées : chevauchées.
            clipped (int): Rejetées : écrêtées.
            low_quality (int): Rejetées avant la règle (qualité insuffisante) ; toujours 0 sous une règle.
            persons_with_reference (int): Personnes dont au moins un Échantillon vocal validé actif est contributif : elles
                ont une Référence.
     """

    present: int
    comparable: int
    contributive: int
    too_short: int
    overlap_only: int
    clipped: int
    low_quality: int
    persons_with_reference: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        present = self.present

        comparable = self.comparable

        contributive = self.contributive

        too_short = self.too_short

        overlap_only = self.overlap_only

        clipped = self.clipped

        low_quality = self.low_quality

        persons_with_reference = self.persons_with_reference


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "present": present,
            "comparable": comparable,
            "contributive": contributive,
            "too_short": too_short,
            "overlap_only": overlap_only,
            "clipped": clipped,
            "low_quality": low_quality,
            "persons_with_reference": persons_with_reference,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        present = d.pop("present")

        comparable = d.pop("comparable")

        contributive = d.pop("contributive")

        too_short = d.pop("too_short")

        overlap_only = d.pop("overlap_only")

        clipped = d.pop("clipped")

        low_quality = d.pop("low_quality")

        persons_with_reference = d.pop("persons_with_reference")

        voice_eligibility_outcome = cls(
            present=present,
            comparable=comparable,
            contributive=contributive,
            too_short=too_short,
            overlap_only=overlap_only,
            clipped=clipped,
            low_quality=low_quality,
            persons_with_reference=persons_with_reference,
        )


        voice_eligibility_outcome.additional_properties = d
        return voice_eligibility_outcome

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
