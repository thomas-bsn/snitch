from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.clip_reanalysis_stake_corrections_type_0 import ClipReanalysisStakeCorrectionsType0
  from ..models.clip_reanalysis_stake_voice_type_0 import ClipReanalysisStakeVoiceType0





T = TypeVar("T", bound="ClipReanalysisStake")



@_attrs_define
class ClipReanalysisStake:
    """ Ce que la réanalyse rendrait obsolète ; rien n'est supprimé (historiques, échantillons validés et anciens résultats
    conservés).

        Attributes:
            corrections (ClipReanalysisStakeCorrectionsType0 | None): Corrections de segments de la transcription visible
                (retranscription).
            voice (ClipReanalysisStakeVoiceType0 | None): Validations de voix, échantillons et corrections de locuteur
                (nouvelle diarisation).
     """

    corrections: ClipReanalysisStakeCorrectionsType0 | None
    voice: ClipReanalysisStakeVoiceType0 | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_reanalysis_stake_corrections_type_0 import ClipReanalysisStakeCorrectionsType0 # noqa: PLC0415
        from ..models.clip_reanalysis_stake_voice_type_0 import ClipReanalysisStakeVoiceType0 # noqa: PLC0415
        corrections: dict[str, Any] | None
        if isinstance(self.corrections, ClipReanalysisStakeCorrectionsType0):
            corrections = self.corrections.to_dict()
        else:
            corrections = self.corrections

        voice: dict[str, Any] | None
        if isinstance(self.voice, ClipReanalysisStakeVoiceType0):
            voice = self.voice.to_dict()
        else:
            voice = self.voice


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "corrections": corrections,
            "voice": voice,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_reanalysis_stake_corrections_type_0 import ClipReanalysisStakeCorrectionsType0 # noqa: PLC0415
        from ..models.clip_reanalysis_stake_voice_type_0 import ClipReanalysisStakeVoiceType0 # noqa: PLC0415
        d = dict(src_dict)
        def _parse_corrections(data: object) -> ClipReanalysisStakeCorrectionsType0 | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                corrections_type_0 = ClipReanalysisStakeCorrectionsType0.from_dict(data)



                return corrections_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(ClipReanalysisStakeCorrectionsType0 | None, data)

        corrections = _parse_corrections(d.pop("corrections"))


        def _parse_voice(data: object) -> ClipReanalysisStakeVoiceType0 | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                voice_type_0 = ClipReanalysisStakeVoiceType0.from_dict(data)



                return voice_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(ClipReanalysisStakeVoiceType0 | None, data)

        voice = _parse_voice(d.pop("voice"))


        clip_reanalysis_stake = cls(
            corrections=corrections,
            voice=voice,
        )


        clip_reanalysis_stake.additional_properties = d
        return clip_reanalysis_stake

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
