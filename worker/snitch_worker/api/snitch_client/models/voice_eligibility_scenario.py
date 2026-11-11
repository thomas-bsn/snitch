from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_eligibility_input import VoiceEligibilityInput





T = TypeVar("T", bound="VoiceEligibilityScenario")



@_attrs_define
class VoiceEligibilityScenario:
    """ Éligibilité qu'une requalification sous cette règle écrirait, comparée à l'état enregistré.

        Attributes:
            rule (VoiceEligibilityInput):
            present (int): Empreintes de Locuteur local de production présentes.
            comparable (int): Comparables.
            contributive (int): Contributives (comparables comprises).
            too_short (int): Ni comparables ni contributives : Parole nette trop courte.
            overlap_only (int): Rejetées : chevauchées.
            clipped (int): Rejetées : écrêtées.
            low_quality (int): Rejetées avant la règle (qualité insuffisante) ; toujours 0 sous une règle.
            persons_with_reference (int): Personnes dont au moins un Échantillon vocal validé actif est contributif : elles
                ont une Référence.
            recovered_comparable (int): Empreintes qui deviendraient comparables.
            recovered_contributive (int): Empreintes qui deviendraient contributives.
            lost_comparable (int): Empreintes comparables aujourd'hui qui cesseraient de l'être.
            lost_contributive (int): Empreintes contributives aujourd'hui qui cesseraient de l'être.
     """

    rule: VoiceEligibilityInput
    present: int
    comparable: int
    contributive: int
    too_short: int
    overlap_only: int
    clipped: int
    low_quality: int
    persons_with_reference: int
    recovered_comparable: int
    recovered_contributive: int
    lost_comparable: int
    lost_contributive: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_eligibility_input import VoiceEligibilityInput # noqa: PLC0415
        rule = self.rule.to_dict()

        present = self.present

        comparable = self.comparable

        contributive = self.contributive

        too_short = self.too_short

        overlap_only = self.overlap_only

        clipped = self.clipped

        low_quality = self.low_quality

        persons_with_reference = self.persons_with_reference

        recovered_comparable = self.recovered_comparable

        recovered_contributive = self.recovered_contributive

        lost_comparable = self.lost_comparable

        lost_contributive = self.lost_contributive


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "rule": rule,
            "present": present,
            "comparable": comparable,
            "contributive": contributive,
            "too_short": too_short,
            "overlap_only": overlap_only,
            "clipped": clipped,
            "low_quality": low_quality,
            "persons_with_reference": persons_with_reference,
            "recovered_comparable": recovered_comparable,
            "recovered_contributive": recovered_contributive,
            "lost_comparable": lost_comparable,
            "lost_contributive": lost_contributive,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_eligibility_input import VoiceEligibilityInput # noqa: PLC0415
        d = dict(src_dict)
        rule = VoiceEligibilityInput.from_dict(d.pop("rule"))




        present = d.pop("present")

        comparable = d.pop("comparable")

        contributive = d.pop("contributive")

        too_short = d.pop("too_short")

        overlap_only = d.pop("overlap_only")

        clipped = d.pop("clipped")

        low_quality = d.pop("low_quality")

        persons_with_reference = d.pop("persons_with_reference")

        recovered_comparable = d.pop("recovered_comparable")

        recovered_contributive = d.pop("recovered_contributive")

        lost_comparable = d.pop("lost_comparable")

        lost_contributive = d.pop("lost_contributive")

        voice_eligibility_scenario = cls(
            rule=rule,
            present=present,
            comparable=comparable,
            contributive=contributive,
            too_short=too_short,
            overlap_only=overlap_only,
            clipped=clipped,
            low_quality=low_quality,
            persons_with_reference=persons_with_reference,
            recovered_comparable=recovered_comparable,
            recovered_contributive=recovered_contributive,
            lost_comparable=lost_comparable,
            lost_contributive=lost_contributive,
        )


        voice_eligibility_scenario.additional_properties = d
        return voice_eligibility_scenario

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
