from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_trace_fingerprint_scope import VoiceTraceFingerprintScope
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
            usable (bool):
            unusable_reason (str):
            model (str):
            model_version (str):
            measures (FingerprintMeasures | None): Null pour une empreinte enregistrée sans mesures (Worker ou ligne
                antérieurs) : mesures inconnues.
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
    unusable_reason: str
    model: str
    model_version: str
    measures: FingerprintMeasures | None
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

        unusable_reason = self.unusable_reason

        model = self.model

        model_version = self.model_version

        measures: dict[str, Any] | None
        if isinstance(self.measures, FingerprintMeasures):
            measures = self.measures.to_dict()
        else:
            measures = self.measures


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

        unusable_reason = d.pop("unusable_reason")

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
