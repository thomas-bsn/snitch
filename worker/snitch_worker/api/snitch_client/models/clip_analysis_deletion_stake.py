from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.clip_analysis_deletion_stake_corrections_type_0 import ClipAnalysisDeletionStakeCorrectionsType0





T = TypeVar("T", bound="ClipAnalysisDeletionStake")



@_attrs_define
class ClipAnalysisDeletionStake:
    """ Pertes d'une suppression d'analyse ; rien n'est supprimé par ce calcul. Les Personnes et leurs échantillons validés
    ne sont jamais supprimés, seulement retirés de la référence.

        Attributes:
            segments (int): Segments des transcriptions supprimées.
            corrections (ClipAnalysisDeletionStakeCorrectionsType0 | None): Corrections de segments de la transcription
                visible (retranscription).
            voices (int): Locuteurs locaux du clip (diarisation).
            validated_voices (int): Locuteurs locaux validés par un humain.
            voice_samples (int): Échantillons actifs de la référence retirés.
            persons (int): Personnes dont une voix validée ou un échantillon actif est touché.
     """

    segments: int
    corrections: ClipAnalysisDeletionStakeCorrectionsType0 | None
    voices: int
    validated_voices: int
    voice_samples: int
    persons: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_deletion_stake_corrections_type_0 import ClipAnalysisDeletionStakeCorrectionsType0 # noqa: PLC0415
        segments = self.segments

        corrections: dict[str, Any] | None
        if isinstance(self.corrections, ClipAnalysisDeletionStakeCorrectionsType0):
            corrections = self.corrections.to_dict()
        else:
            corrections = self.corrections

        voices = self.voices

        validated_voices = self.validated_voices

        voice_samples = self.voice_samples

        persons = self.persons


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "segments": segments,
            "corrections": corrections,
            "voices": voices,
            "validated_voices": validated_voices,
            "voice_samples": voice_samples,
            "persons": persons,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_deletion_stake_corrections_type_0 import ClipAnalysisDeletionStakeCorrectionsType0 # noqa: PLC0415
        d = dict(src_dict)
        segments = d.pop("segments")

        def _parse_corrections(data: object) -> ClipAnalysisDeletionStakeCorrectionsType0 | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                corrections_type_0 = ClipAnalysisDeletionStakeCorrectionsType0.from_dict(data)



                return corrections_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(ClipAnalysisDeletionStakeCorrectionsType0 | None, data)

        corrections = _parse_corrections(d.pop("corrections"))


        voices = d.pop("voices")

        validated_voices = d.pop("validated_voices")

        voice_samples = d.pop("voice_samples")

        persons = d.pop("persons")

        clip_analysis_deletion_stake = cls(
            segments=segments,
            corrections=corrections,
            voices=voices,
            validated_voices=validated_voices,
            voice_samples=voice_samples,
            persons=persons,
        )


        clip_analysis_deletion_stake.additional_properties = d
        return clip_analysis_deletion_stake

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
