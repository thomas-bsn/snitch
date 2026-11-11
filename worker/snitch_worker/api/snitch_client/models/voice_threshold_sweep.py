from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="VoiceThresholdSweep")



@_attrs_define
class VoiceThresholdSweep:
    """ Décision rejouée sur les tests à un jeu de seuils.

        Attributes:
            automatic_threshold (float):
            suggestion_threshold (float): Seuil de suggestion utilisé : celui du groupe, abaissé au seuil automatique s'il
                le dépasse.
            ambiguity_margin (float):
            automatic (int): Tests attribués automatiquement.
            errors (int): Attributions automatiques désignant une autre Personne que celle validée.
            to_verify (int): Tests laissés à vérifier (suggestion, ambiguïté ou inconnu).
            unknown_automatic (int): Inconnus simulés (tests rejoués sans la Référence de leur Personne) reconnus à tort
                automatiquement.
            unknown_automatic_rate (float | None): Taux d'inconnus simulés reconnus à tort : unknown_automatic sur le nombre
                d'inconnus simulés rejoués (un par test) ; null sans test.
     """

    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    automatic: int
    errors: int
    to_verify: int
    unknown_automatic: int
    unknown_automatic_rate: float | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        automatic_threshold = self.automatic_threshold

        suggestion_threshold = self.suggestion_threshold

        ambiguity_margin = self.ambiguity_margin

        automatic = self.automatic

        errors = self.errors

        to_verify = self.to_verify

        unknown_automatic = self.unknown_automatic

        unknown_automatic_rate: float | None
        unknown_automatic_rate = self.unknown_automatic_rate


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "automatic_threshold": automatic_threshold,
            "suggestion_threshold": suggestion_threshold,
            "ambiguity_margin": ambiguity_margin,
            "automatic": automatic,
            "errors": errors,
            "to_verify": to_verify,
            "unknown_automatic": unknown_automatic,
            "unknown_automatic_rate": unknown_automatic_rate,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automatic_threshold = d.pop("automatic_threshold")

        suggestion_threshold = d.pop("suggestion_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        automatic = d.pop("automatic")

        errors = d.pop("errors")

        to_verify = d.pop("to_verify")

        unknown_automatic = d.pop("unknown_automatic")

        def _parse_unknown_automatic_rate(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        unknown_automatic_rate = _parse_unknown_automatic_rate(d.pop("unknown_automatic_rate"))


        voice_threshold_sweep = cls(
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            automatic=automatic,
            errors=errors,
            to_verify=to_verify,
            unknown_automatic=unknown_automatic,
            unknown_automatic_rate=unknown_automatic_rate,
        )


        voice_threshold_sweep.additional_properties = d
        return voice_threshold_sweep

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
