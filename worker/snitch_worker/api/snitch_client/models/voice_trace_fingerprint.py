from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_trace_fingerprint_net_speech_source import VoiceTraceFingerprintNetSpeechSource
from ..models.voice_trace_fingerprint_scope import VoiceTraceFingerprintScope
from ..models.voice_trace_fingerprint_unusable_reason import VoiceTraceFingerprintUnusableReason
from typing import cast

if TYPE_CHECKING:
  from ..models.fingerprint_measures import FingerprintMeasures





T = TypeVar("T", bound="VoiceTraceFingerprint")



@_attrs_define
class VoiceTraceFingerprint:
    """
        Attributes:
            id (int):
            local_speaker_id (int):
            label (str):
            scope (VoiceTraceFingerprintScope):
            turn_index (int | None): Tour de l'empreinte ; null pour l'empreinte d'un Locuteur local.
            start_seconds (float):
            end_seconds (float):
            duration_seconds (float):
            quality (float):
            usable (bool): Comparable : rapprochée seule d'une Référence. Décidé par la règle d'éligibilité (ADR 0064).
            unusable_reason (VoiceTraceFingerprintUnusableReason): Pourquoi l'empreinte n'est pas comparable (vide si elle
                l'est). too_short : Parole nette sous la durée comparable de la règle (elle peut rester contributive, voir
                contributive) ; overlap_only : aucun intervalle net suffisant, l'empreinte est faite de tours chevauchés ;
                clipped : trop d'échantillons écrêtés ; low_quality : décidée avant la règle d'éligibilité (empreintes
                antérieures ou de benchmark).
            model (str):
            model_version (str):
            measures (FingerprintMeasures | None): Null pour une empreinte enregistrée sans mesures (Worker ou ligne
                antérieurs) : mesures inconnues.
            contributive (bool): Peut entrer dans une Référence une fois validée par un humain. Une empreinte comparable
                (usable) est toujours contributive ; une empreinte chevauchée ou écrêtée ne l'est jamais.
            eligibility_rule_version (int | None): Version de la règle d'éligibilité qui a décidé. Null : décidée à
                l'enregistrement (empreinte antérieure à la règle, jusqu'à une requalification).
            net_speech_seconds (float | None): Parole nette retenue par la règle. Null : pas encore évaluée.
            net_speech_source (VoiceTraceFingerprintNetSpeechSource): Origine de la Parole nette : worker (mesures du
                Worker), turns (calculée depuis les tours stockés), estimated (ni mesures ni tours, par exemple un Locuteur
                local validé puis archivé : durée de l'empreinte, majorant de la Parole nette).
     """

    id: int
    local_speaker_id: int
    label: str
    scope: VoiceTraceFingerprintScope
    turn_index: int | None
    start_seconds: float
    end_seconds: float
    duration_seconds: float
    quality: float
    usable: bool
    unusable_reason: VoiceTraceFingerprintUnusableReason
    model: str
    model_version: str
    measures: FingerprintMeasures | None
    contributive: bool
    eligibility_rule_version: int | None
    net_speech_seconds: float | None
    net_speech_source: VoiceTraceFingerprintNetSpeechSource
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.fingerprint_measures import FingerprintMeasures # noqa: PLC0415
        id = self.id

        local_speaker_id = self.local_speaker_id

        label = self.label

        scope = self.scope.value

        turn_index: int | None
        turn_index = self.turn_index

        start_seconds = self.start_seconds

        end_seconds = self.end_seconds

        duration_seconds = self.duration_seconds

        quality = self.quality

        usable = self.usable

        unusable_reason = self.unusable_reason.value

        model = self.model

        model_version = self.model_version

        measures: dict[str, Any] | None
        if isinstance(self.measures, FingerprintMeasures):
            measures = self.measures.to_dict()
        else:
            measures = self.measures

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
            "local_speaker_id": local_speaker_id,
            "label": label,
            "scope": scope,
            "turn_index": turn_index,
            "start_seconds": start_seconds,
            "end_seconds": end_seconds,
            "duration_seconds": duration_seconds,
            "quality": quality,
            "usable": usable,
            "unusable_reason": unusable_reason,
            "model": model,
            "model_version": model_version,
            "measures": measures,
            "contributive": contributive,
            "eligibility_rule_version": eligibility_rule_version,
            "net_speech_seconds": net_speech_seconds,
            "net_speech_source": net_speech_source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.fingerprint_measures import FingerprintMeasures # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        local_speaker_id = d.pop("local_speaker_id")

        label = d.pop("label")

        scope = VoiceTraceFingerprintScope(d.pop("scope"))




        def _parse_turn_index(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        turn_index = _parse_turn_index(d.pop("turn_index"))


        start_seconds = d.pop("start_seconds")

        end_seconds = d.pop("end_seconds")

        duration_seconds = d.pop("duration_seconds")

        quality = d.pop("quality")

        usable = d.pop("usable")

        unusable_reason = VoiceTraceFingerprintUnusableReason(d.pop("unusable_reason"))




        model = d.pop("model")

        model_version = d.pop("model_version")

        def _parse_measures(data: object) -> FingerprintMeasures | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                measures_type_1 = FingerprintMeasures.from_dict(data)



                return measures_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(FingerprintMeasures | None, data)

        measures = _parse_measures(d.pop("measures"))


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


        net_speech_source = VoiceTraceFingerprintNetSpeechSource(d.pop("net_speech_source"))




        voice_trace_fingerprint = cls(
            id=id,
            local_speaker_id=local_speaker_id,
            label=label,
            scope=scope,
            turn_index=turn_index,
            start_seconds=start_seconds,
            end_seconds=end_seconds,
            duration_seconds=duration_seconds,
            quality=quality,
            usable=usable,
            unusable_reason=unusable_reason,
            model=model,
            model_version=model_version,
            measures=measures,
            contributive=contributive,
            eligibility_rule_version=eligibility_rule_version,
            net_speech_seconds=net_speech_seconds,
            net_speech_source=net_speech_source,
        )


        voice_trace_fingerprint.additional_properties = d
        return voice_trace_fingerprint

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
