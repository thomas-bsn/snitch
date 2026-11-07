from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AudioPlanVoiceStake")



@_attrs_define
class AudioPlanVoiceStake:
    """ Travail humain qu'une diarisation des pistes concernées rendrait obsolète. Les anciennes voix restent protégées
    jusqu'à un nouveau résultat réussi.

        Attributes:
            validated_speakers (int):
            voice_samples (int):
            speaker_corrections (int):
            affected_persons (int): Personnes dont la Référence perd un échantillon vocal ou une validation.
            audio_stream_indices (list[int]): Pistes qui portent ce travail humain.
     """

    validated_speakers: int
    voice_samples: int
    speaker_corrections: int
    affected_persons: int
    audio_stream_indices: list[int]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        validated_speakers = self.validated_speakers

        voice_samples = self.voice_samples

        speaker_corrections = self.speaker_corrections

        affected_persons = self.affected_persons

        audio_stream_indices = self.audio_stream_indices




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "validated_speakers": validated_speakers,
            "voice_samples": voice_samples,
            "speaker_corrections": speaker_corrections,
            "affected_persons": affected_persons,
            "audio_stream_indices": audio_stream_indices,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        validated_speakers = d.pop("validated_speakers")

        voice_samples = d.pop("voice_samples")

        speaker_corrections = d.pop("speaker_corrections")

        affected_persons = d.pop("affected_persons")

        audio_stream_indices = cast(list[int], d.pop("audio_stream_indices"))


        audio_plan_voice_stake = cls(
            validated_speakers=validated_speakers,
            voice_samples=voice_samples,
            speaker_corrections=speaker_corrections,
            affected_persons=affected_persons,
            audio_stream_indices=audio_stream_indices,
        )


        audio_plan_voice_stake.additional_properties = d
        return audio_plan_voice_stake

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
