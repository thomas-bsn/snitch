from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

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
     """

    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    max_candidates: int
    group: VoiceFingerprintGroup | Unset = UNSET
    source: VoiceMatchingInputSource | Unset = UNSET





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




        voice_matching_input = cls(
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            max_candidates=max_candidates,
            group=group,
            source=source,
        )

        return voice_matching_input
