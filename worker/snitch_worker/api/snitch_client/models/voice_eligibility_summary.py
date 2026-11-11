from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceEligibilitySummary")



@_attrs_define
class VoiceEligibilitySummary:
    """ Empreintes de Locuteur local de production, par éligibilité.

        Attributes:
            comparable (int): Comparables.
            contributive_only (int): Contributives sans être comparables.
            too_short (int): Ni comparables ni contributives : Parole nette trop courte.
            overlap_only (int): Rejetées : chevauchées.
            clipped (int): Rejetées : écrêtées.
            low_quality (int): Rejetées avant la règle (qualité insuffisante).
            stale (int): Décidées par une autre règle que la règle active (ou avant la règle).
     """

    comparable: int
    contributive_only: int
    too_short: int
    overlap_only: int
    clipped: int
    low_quality: int
    stale: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        comparable = self.comparable

        contributive_only = self.contributive_only

        too_short = self.too_short

        overlap_only = self.overlap_only

        clipped = self.clipped

        low_quality = self.low_quality

        stale = self.stale


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "comparable": comparable,
            "contributive_only": contributive_only,
            "too_short": too_short,
            "overlap_only": overlap_only,
            "clipped": clipped,
            "low_quality": low_quality,
            "stale": stale,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        comparable = d.pop("comparable")

        contributive_only = d.pop("contributive_only")

        too_short = d.pop("too_short")

        overlap_only = d.pop("overlap_only")

        clipped = d.pop("clipped")

        low_quality = d.pop("low_quality")

        stale = d.pop("stale")

        voice_eligibility_summary = cls(
            comparable=comparable,
            contributive_only=contributive_only,
            too_short=too_short,
            overlap_only=overlap_only,
            clipped=clipped,
            low_quality=low_quality,
            stale=stale,
        )


        voice_eligibility_summary.additional_properties = d
        return voice_eligibility_summary

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
