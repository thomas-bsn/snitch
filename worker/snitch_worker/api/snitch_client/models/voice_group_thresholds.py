from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_group_thresholds_score_mode import VoiceGroupThresholdsScoreMode
from ..models.voice_group_thresholds_source import VoiceGroupThresholdsSource
from typing import cast
import datetime






T = TypeVar("T", bound="VoiceGroupThresholds")



@_attrs_define
class VoiceGroupThresholds:
    """ Une version des seuils appliqués à un groupe d'empreintes (ADR 0064). La plus haute version est en vigueur.

        Attributes:
            version (int):
            suggestion_threshold (float):
            automatic_threshold (float):
            ambiguity_margin (float):
            source (VoiceGroupThresholdsSource): migration : seuils de l'installation repris pour le groupe actif ;
                administrator : saisis ; proposal : proposition de la Précision mesurée appliquée par l'administrateur.
            score_mode (VoiceGroupThresholdsScoreMode): Mode de score actif au moment de l'application.
            author (None | str): Administrateur ; null pour Snitch (migration).
            created_at (datetime.datetime):
     """

    version: int
    suggestion_threshold: float
    automatic_threshold: float
    ambiguity_margin: float
    source: VoiceGroupThresholdsSource
    score_mode: VoiceGroupThresholdsScoreMode
    author: None | str
    created_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        suggestion_threshold = self.suggestion_threshold

        automatic_threshold = self.automatic_threshold

        ambiguity_margin = self.ambiguity_margin

        source = self.source.value

        score_mode = self.score_mode.value

        author: None | str
        author = self.author

        created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "suggestion_threshold": suggestion_threshold,
            "automatic_threshold": automatic_threshold,
            "ambiguity_margin": ambiguity_margin,
            "source": source,
            "score_mode": score_mode,
            "author": author,
            "created_at": created_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        suggestion_threshold = d.pop("suggestion_threshold")

        automatic_threshold = d.pop("automatic_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        source = VoiceGroupThresholdsSource(d.pop("source"))




        score_mode = VoiceGroupThresholdsScoreMode(d.pop("score_mode"))




        def _parse_author(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        author = _parse_author(d.pop("author"))


        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        voice_group_thresholds = cls(
            version=version,
            suggestion_threshold=suggestion_threshold,
            automatic_threshold=automatic_threshold,
            ambiguity_margin=ambiguity_margin,
            source=source,
            score_mode=score_mode,
            author=author,
            created_at=created_at,
        )


        voice_group_thresholds.additional_properties = d
        return voice_group_thresholds

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
