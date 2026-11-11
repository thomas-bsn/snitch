from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_matching_input_score_mode import VoiceMatchingInputScoreMode
from ..models.voice_matching_input_source import VoiceMatchingInputSource
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_fingerprint_group import VoiceFingerprintGroup





T = TypeVar("T", bound="VoiceMatchingInput")



@_attrs_define
class VoiceMatchingInput:
    """
        Attributes:
            automatic_threshold (float):
            suggestion_threshold (float): Au plus égal au seuil automatique.
            ambiguity_margin (float):
            max_candidates (int):
            group (VoiceFingerprintGroup | Unset): Groupe d'empreintes (modèle, version, dimensions) : seules les empreintes
                d'un même groupe sont comparées.
            source (VoiceMatchingInputSource | Unset): Origine tracée des seuils appliqués ; administrator par défaut.
                proposal n'est accepté que pour les valeurs exactes de la proposition actuelle de la mesure pour ce groupe et ce
                mode de score, sinon 409 voice_threshold_proposal_stale.
            score_mode (VoiceMatchingInputScoreMode | Unset): Mode de score pour lequel les seuils sont appliqués ; mode
                actif par défaut. Les seuils d'un mode ne valent jamais pour l'autre.
     """

    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    max_candidates: int
    group: VoiceFingerprintGroup | Unset = UNSET
    source: VoiceMatchingInputSource | Unset = UNSET
    score_mode: VoiceMatchingInputScoreMode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        automatic_threshold = self.automatic_threshold

        suggestion_threshold = self.suggestion_threshold

        ambiguity_margin = self.ambiguity_margin

        max_candidates = self.max_candidates

        group: dict[str, Any] | Unset = UNSET
        if not isinstance(self.group, Unset):
            group = self.group.to_dict()

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value


        score_mode: str | Unset = UNSET
        if not isinstance(self.score_mode, Unset):
            score_mode = self.score_mode.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automatic_threshold": automatic_threshold,
            "suggestion_threshold": suggestion_threshold,
            "ambiguity_margin": ambiguity_margin,
            "max_candidates": max_candidates,
        })
        if group is not UNSET:
            field_dict["group"] = group
        if source is not UNSET:
            field_dict["source"] = source
        if score_mode is not UNSET:
            field_dict["score_mode"] = score_mode

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        d = dict(src_dict)
        automatic_threshold = d.pop("automatic_threshold")

        suggestion_threshold = d.pop("suggestion_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        max_candidates = d.pop("max_candidates")

        _group = d.pop("group", UNSET)
        group: VoiceFingerprintGroup | Unset
        if isinstance(_group,  Unset):
            group = UNSET
        else:
            group = VoiceFingerprintGroup.from_dict(_group)




        _source = d.pop("source", UNSET)
        source: VoiceMatchingInputSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = VoiceMatchingInputSource(_source)




        _score_mode = d.pop("score_mode", UNSET)
        score_mode: VoiceMatchingInputScoreMode | Unset
        if isinstance(_score_mode,  Unset):
            score_mode = UNSET
        else:
            score_mode = VoiceMatchingInputScoreMode(_score_mode)




        voice_matching_input = cls(
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            max_candidates=max_candidates,
            group=group,
            source=source,
            score_mode=score_mode,
        )

        return voice_matching_input
