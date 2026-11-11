from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="VoiceEligibilityPersonPreview")



@_attrs_define
class VoiceEligibilityPersonPreview:
    """ Échantillons validés actifs d'une Personne dans un groupe d'empreintes, sous la règle examinée.

        Attributes:
            person (PersonRef):
            present (int): Empreintes de ses échantillons validés actifs.
            comparable (int): Comparables sous la règle examinée.
            contributive (int): Contributives sous la règle examinée.
            too_short (int): Trop courtes, non contributives.
            overlap_only (int): Chevauchées.
            clipped (int): Écrêtées.
            missing_sources (int): Dont le clip n'a plus de source audio connue en base.
            estimated (int): Dont la Parole nette est estimée (ni mesures ni tours).
            reference (bool): A une Référence sous la règle examinée.
            reference_now (bool): A une Référence selon l'éligibilité enregistrée.
     """

    person: PersonRef
    present: int
    comparable: int
    contributive: int
    too_short: int
    overlap_only: int
    clipped: int
    missing_sources: int
    estimated: int
    reference: bool
    reference_now: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        person = self.person.to_dict()

        present = self.present

        comparable = self.comparable

        contributive = self.contributive

        too_short = self.too_short

        overlap_only = self.overlap_only

        clipped = self.clipped

        missing_sources = self.missing_sources

        estimated = self.estimated

        reference = self.reference

        reference_now = self.reference_now


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "person": person,
            "present": present,
            "comparable": comparable,
            "contributive": contributive,
            "too_short": too_short,
            "overlap_only": overlap_only,
            "clipped": clipped,
            "missing_sources": missing_sources,
            "estimated": estimated,
            "reference": reference,
            "reference_now": reference_now,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        person = PersonRef.from_dict(d.pop("person"))




        present = d.pop("present")

        comparable = d.pop("comparable")

        contributive = d.pop("contributive")

        too_short = d.pop("too_short")

        overlap_only = d.pop("overlap_only")

        clipped = d.pop("clipped")

        missing_sources = d.pop("missing_sources")

        estimated = d.pop("estimated")

        reference = d.pop("reference")

        reference_now = d.pop("reference_now")

        voice_eligibility_person_preview = cls(
            person=person,
            present=present,
            comparable=comparable,
            contributive=contributive,
            too_short=too_short,
            overlap_only=overlap_only,
            clipped=clipped,
            missing_sources=missing_sources,
            estimated=estimated,
            reference=reference,
            reference_now=reference_now,
        )


        voice_eligibility_person_preview.additional_properties = d
        return voice_eligibility_person_preview

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
