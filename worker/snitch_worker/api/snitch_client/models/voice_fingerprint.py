from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_fingerprint_net_speech_source import VoiceFingerprintNetSpeechSource
from ..models.voice_fingerprint_scope import VoiceFingerprintScope
from ..models.voice_fingerprint_unusable_reason import VoiceFingerprintUnusableReason
from typing import cast






T = TypeVar("T", bound="VoiceFingerprint")



@_attrs_define
class VoiceFingerprint:
    """
        Attributes:
            id (int):
            scope (VoiceFingerprintScope):
            start_seconds (float):
            end_seconds (float):
            duration_seconds (float):
            quality (float):
            usable (bool): Comparable : rapprochée seule d'une Référence. Décidé par la règle d'éligibilité (ADR 0064).
            unusable_reason (VoiceFingerprintUnusableReason): Pourquoi l'empreinte n'est pas comparable (vide si elle
                l'est). too_short : Parole nette sous la durée comparable de la règle (elle peut rester contributive, voir
                contributive) ; overlap_only : aucun intervalle net suffisant, l'empreinte est faite de tours chevauchés ;
                clipped : trop d'échantillons écrêtés ; low_quality : décidée avant la règle d'éligibilité (empreintes
                antérieures ou de benchmark).
            model (str):
            model_version (str):
            dimensions (int):
            contributive (bool): Peut entrer dans une Référence une fois validée par un humain. Une empreinte comparable
                (usable) est toujours contributive ; une empreinte chevauchée ou écrêtée ne l'est jamais.
            eligibility_rule_version (int | None): Version de la règle d'éligibilité qui a décidé. Null : décidée à
                l'enregistrement (empreinte antérieure à la règle, jusqu'à une requalification).
            net_speech_seconds (float | None): Parole nette retenue par la règle. Null : pas encore évaluée.
            net_speech_source (VoiceFingerprintNetSpeechSource): Origine de la Parole nette : worker (mesures du Worker),
                turns (calculée depuis les tours stockés), estimated (ni mesures ni tours, par exemple un Locuteur local validé
                puis archivé : durée de l'empreinte, majorant de la Parole nette).
     """

    id: int
    scope: VoiceFingerprintScope
    start_seconds: float
    end_seconds: float
    duration_seconds: float
    quality: float
    usable: bool
    unusable_reason: VoiceFingerprintUnusableReason
    model: str
    model_version: str
    dimensions: int
    contributive: bool
    eligibility_rule_version: int | None
    net_speech_seconds: float | None
    net_speech_source: VoiceFingerprintNetSpeechSource
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        scope = self.scope.value

        start_seconds = self.start_seconds

        end_seconds = self.end_seconds

        duration_seconds = self.duration_seconds

        quality = self.quality

        usable = self.usable

        unusable_reason = self.unusable_reason.value

        model = self.model

        model_version = self.model_version

        dimensions = self.dimensions

        contributive = self.contributive

        eligibility_rule_version: int | None
        eligibility_rule_version = self.eligibility_rule_version

        net_speech_seconds: float | None
        net_speech_seconds = self.net_speech_seconds

        net_speech_source = self.net_speech_source.value


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "scope": scope,
            "start_seconds": start_seconds,
            "end_seconds": end_seconds,
            "duration_seconds": duration_seconds,
            "quality": quality,
            "usable": usable,
            "unusable_reason": unusable_reason,
            "model": model,
            "model_version": model_version,
            "dimensions": dimensions,
            "contributive": contributive,
            "eligibility_rule_version": eligibility_rule_version,
            "net_speech_seconds": net_speech_seconds,
            "net_speech_source": net_speech_source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        scope = VoiceFingerprintScope(d.pop("scope"))




        start_seconds = d.pop("start_seconds")

        end_seconds = d.pop("end_seconds")

        duration_seconds = d.pop("duration_seconds")

        quality = d.pop("quality")

        usable = d.pop("usable")

        unusable_reason = VoiceFingerprintUnusableReason(d.pop("unusable_reason"))




        model = d.pop("model")

        model_version = d.pop("model_version")

        dimensions = d.pop("dimensions")

        contributive = d.pop("contributive")

        def _parse_eligibility_rule_version(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        eligibility_rule_version = _parse_eligibility_rule_version(d.pop("eligibility_rule_version"))


        def _parse_net_speech_seconds(data: object) -> float | None:
            if data is None:
                return data
            return cast(float | None, data)

        net_speech_seconds = _parse_net_speech_seconds(d.pop("net_speech_seconds"))


        net_speech_source = VoiceFingerprintNetSpeechSource(d.pop("net_speech_source"))




        voice_fingerprint = cls(
            id=id,
            scope=scope,
            start_seconds=start_seconds,
            end_seconds=end_seconds,
            duration_seconds=duration_seconds,
            quality=quality,
            usable=usable,
            unusable_reason=unusable_reason,
            model=model,
            model_version=model_version,
            dimensions=dimensions,
            contributive=contributive,
            eligibility_rule_version=eligibility_rule_version,
            net_speech_seconds=net_speech_seconds,
            net_speech_source=net_speech_source,
        )


        voice_fingerprint.additional_properties = d
        return voice_fingerprint

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
