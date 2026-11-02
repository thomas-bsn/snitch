from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ClipReanalysisStakeVoiceType0")



@_attrs_define
class ClipReanalysisStakeVoiceType0:
    """ Validations de voix, échantillons et corrections de locuteur (nouvelle diarisation).

        Attributes:
            validated_speakers (int):
            voice_samples (int):
            speaker_corrections (int):
     """

    validated_speakers: int
    voice_samples: int
    speaker_corrections: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        validated_speakers = self.validated_speakers

        voice_samples = self.voice_samples

        speaker_corrections = self.speaker_corrections


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "validated_speakers": validated_speakers,
            "voice_samples": voice_samples,
            "speaker_corrections": speaker_corrections,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        validated_speakers = d.pop("validated_speakers")

        voice_samples = d.pop("voice_samples")

        speaker_corrections = d.pop("speaker_corrections")

        clip_reanalysis_stake_voice_type_0 = cls(
            validated_speakers=validated_speakers,
            voice_samples=voice_samples,
            speaker_corrections=speaker_corrections,
        )


        clip_reanalysis_stake_voice_type_0.additional_properties = d
        return clip_reanalysis_stake_voice_type_0

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
