from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="VoiceEligibilityRule")



@_attrs_define
class VoiceEligibilityRule:
    """ Une version de la règle d'éligibilité. La version 1 est la règle provisoire de Snitch.

        Attributes:
            version (int):
            comparable_min_seconds (float): Parole nette minimale d'une empreinte comparable.
            contributive_min_seconds (float): Parole nette minimale d'une empreinte contributive ; au plus la durée
                comparable.
            max_clipped_ratio (float): Part maximale d'échantillons écrêtés ; au-delà, l'empreinte est rejetée (clipped).
            author (None | str): Administrateur qui a fixé cette version ; null pour la règle provisoire.
            provisional (bool): Règle provisoire de Snitch, en attente d'une décision de l'administrateur.
            created_at (datetime.datetime):
     """

    version: int
    comparable_min_seconds: float
    contributive_min_seconds: float
    max_clipped_ratio: float
    author: None | str
    provisional: bool
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        comparable_min_seconds = self.comparable_min_seconds

        contributive_min_seconds = self.contributive_min_seconds

        max_clipped_ratio = self.max_clipped_ratio

        author: None | str
        author = self.author

        provisional = self.provisional

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "comparable_min_seconds": comparable_min_seconds,
            "contributive_min_seconds": contributive_min_seconds,
            "max_clipped_ratio": max_clipped_ratio,
            "author": author,
            "provisional": provisional,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        comparable_min_seconds = d.pop("comparable_min_seconds")

        contributive_min_seconds = d.pop("contributive_min_seconds")

        max_clipped_ratio = d.pop("max_clipped_ratio")

        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        provisional = d.pop("provisional")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        voice_eligibility_rule = cls(
            version=version,
            comparable_min_seconds=comparable_min_seconds,
            contributive_min_seconds=contributive_min_seconds,
            max_clipped_ratio=max_clipped_ratio,
            author=author,
            provisional=provisional,
            created_at=created_at,
        )


        voice_eligibility_rule.additional_properties = d
        return voice_eligibility_rule

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
