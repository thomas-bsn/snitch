from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PersonImpact")



@_attrs_define
class PersonImpact:
    """
        Attributes:
            microphone_rules (int): Désignations de piste micro (règles de source et surcharges de clip) qui nomment la
                Personne : déplacées par une fusion, supprimées par une suppression.
            has_hidden_voices (bool): Vrai lorsque la Personne est attribuée sur un clip que l'appelant ne peut pas voir
                (toujours faux pour un administrateur) : une fusion ou une suppression serait refusée
                (person_has_hidden_voices).
            voice_samples (int): Échantillons vocaux validés (actifs et retirés) qu'effacerait la suppression du profil
                vocal.
            fingerprints (int): Empreintes qu'effacerait la suppression du profil vocal.
            attributions (int): Attributions qu'effacerait la suppression du profil vocal ; les voix correspondantes
                redeviendraient inconnues.
     """

    microphone_rules: int
    has_hidden_voices: bool
    voice_samples: int
    fingerprints: int
    attributions: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        microphone_rules = self.microphone_rules

        has_hidden_voices = self.has_hidden_voices

        voice_samples = self.voice_samples

        fingerprints = self.fingerprints

        attributions = self.attributions


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "microphone_rules": microphone_rules,
            "has_hidden_voices": has_hidden_voices,
            "voice_samples": voice_samples,
            "fingerprints": fingerprints,
            "attributions": attributions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        microphone_rules = d.pop("microphone_rules")

        has_hidden_voices = d.pop("has_hidden_voices")

        voice_samples = d.pop("voice_samples")

        fingerprints = d.pop("fingerprints")

        attributions = d.pop("attributions")

        person_impact = cls(
            microphone_rules=microphone_rules,
            has_hidden_voices=has_hidden_voices,
            voice_samples=voice_samples,
            fingerprints=fingerprints,
            attributions=attributions,
        )


        person_impact.additional_properties = d
        return person_impact

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
