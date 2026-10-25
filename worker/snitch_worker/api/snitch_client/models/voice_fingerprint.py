from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_fingerprint_scope import VoiceFingerprintScope
from ..models.voice_fingerprint_unusable_reason import VoiceFingerprintUnusableReason






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
            usable (bool): Faux : ne sert ni de référence ni de base de propagation.
            unusable_reason (VoiceFingerprintUnusableReason):
            model (str):
            model_version (str):
            dimensions (int):
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
