from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRediarizationStake")



@_attrs_define
class VoiceRediarizationStake:
    """ Ce qu'une relance de la diarisation rendrait obsolète : les locuteurs validés (archivés avec leurs échantillons
    vocaux, conservés et liés à l'audio) et les corrections de locuteur de la transcription (obsolètes).

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

        voice_rediarization_stake = cls(
            validated_speakers=validated_speakers,
            voice_samples=voice_samples,
            speaker_corrections=speaker_corrections,
        )


        voice_rediarization_stake.additional_properties = d
        return voice_rediarization_stake

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
