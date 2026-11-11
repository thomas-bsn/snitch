from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_score_mode_input_mode import VoiceScoreModeInputMode






T = TypeVar("T", bound="VoiceScoreModeInput")



@_attrs_define
class VoiceScoreModeInput:
    """
        Attributes:
            mode (VoiceScoreModeInputMode): cumulative : cosinus avec la Référence de la Personne, moyenne de ses
                échantillons contributifs validés pondérée par leur Parole nette plafonnée. best_sample : meilleur cosinus avec
                un seul de ses échantillons comparables.
     """

    mode: VoiceScoreModeInputMode





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = VoiceScoreModeInputMode(d.pop("mode"))




        voice_score_mode_input = cls(
            mode=mode,
        )

        return voice_score_mode_input
