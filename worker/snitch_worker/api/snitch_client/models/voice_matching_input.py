from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceMatchingInput")



@_attrs_define
class VoiceMatchingInput:
    """
        Attributes:
            automatic_threshold (float):
            suggestion_threshold (float): Au plus égal au seuil automatique.
            ambiguity_margin (float):
            max_candidates (int):
     """

    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    max_candidates: int





    def to_dict(self) -> dict[str, Any]:
        automatic_threshold = self.automatic_threshold

        suggestion_threshold = self.suggestion_threshold

        ambiguity_margin = self.ambiguity_margin

        max_candidates = self.max_candidates


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automatic_threshold": automatic_threshold,
            "suggestion_threshold": suggestion_threshold,
            "ambiguity_margin": ambiguity_margin,
            "max_candidates": max_candidates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automatic_threshold = d.pop("automatic_threshold")

        suggestion_threshold = d.pop("suggestion_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        max_candidates = d.pop("max_candidates")

        voice_matching_input = cls(
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            max_candidates=max_candidates,
        )

        return voice_matching_input
