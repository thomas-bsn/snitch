from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="VoiceMatchingSettings")



@_attrs_define
class VoiceMatchingSettings:
    """
        Attributes:
            auto_enabled (bool): Faux par défaut : tant que l'administrateur n'a pas activé l'automatisation, un
                rapprochement n'est jamais plus qu'une suggestion.
            automatic_threshold (float): Similarité minimale d'une attribution automatique.
            suggestion_threshold (float): Similarité minimale d'une suggestion ; en dessous, la voix est inconnue.
            ambiguity_margin (float): Écart sous lequel deux Personnes sont jugées aussi proches : la voix est ambiguë.
            max_candidates (int):
            updated_at (datetime.datetime):
     """

    auto_enabled: bool
    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    max_candidates: int
    updated_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        auto_enabled = self.auto_enabled

        automatic_threshold = self.automatic_threshold

        suggestion_threshold = self.suggestion_threshold

        ambiguity_margin = self.ambiguity_margin

        max_candidates = self.max_candidates

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "auto_enabled": auto_enabled,
            "automatic_threshold": automatic_threshold,
            "suggestion_threshold": suggestion_threshold,
            "ambiguity_margin": ambiguity_margin,
            "max_candidates": max_candidates,
            "updated_at": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        auto_enabled = d.pop("auto_enabled")

        automatic_threshold = d.pop("automatic_threshold")

        suggestion_threshold = d.pop("suggestion_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        max_candidates = d.pop("max_candidates")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        voice_matching_settings = cls(
            auto_enabled=auto_enabled,
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            max_candidates=max_candidates,
            updated_at=updated_at,
        )


        voice_matching_settings.additional_properties = d
        return voice_matching_settings

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
