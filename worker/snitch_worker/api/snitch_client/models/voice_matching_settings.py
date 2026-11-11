from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.voice_fingerprint_group import VoiceFingerprintGroup





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
            precision_threshold (float): Critère de proposition d'activation : précision mesurée minimale. Valeur de départ
                configurable, jamais une garantie.
            min_confirmations (int): Critère de proposition d'activation : nombre minimal d'attributions automatiques
                simulées confirmées par une validation humaine.
            auto_changed_at (datetime.datetime | None): Dernier changement de l'interrupteur d'automatisation ; null s'il
                n'a jamais été modifié.
            updated_at (datetime.datetime):
            group (None | VoiceFingerprintGroup): Groupe d'empreintes actif dont ces seuils sont ceux ; null sur une
                installation sans empreinte (valeurs provisoires).
            thresholds_applied (bool): Le groupe actif a des seuils appliqués par un administrateur. Faux : le groupe ne
                produit que des suggestions, avec les seuils provisoires, et aucune Attribution automatique.
     """

    auto_enabled: bool
    automatic_threshold: float
    suggestion_threshold: float
    ambiguity_margin: float
    max_candidates: int
    precision_threshold: float
    min_confirmations: int
    auto_changed_at: datetime.datetime | None
    updated_at: datetime.datetime
    group: None | VoiceFingerprintGroup
    thresholds_applied: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        auto_enabled = self.auto_enabled

        automatic_threshold = self.automatic_threshold

        suggestion_threshold = self.suggestion_threshold

        ambiguity_margin = self.ambiguity_margin

        max_candidates = self.max_candidates

        precision_threshold = self.precision_threshold

        min_confirmations = self.min_confirmations

        auto_changed_at: None | str
        if isinstance(self.auto_changed_at, datetime.datetime):
            auto_changed_at = self.auto_changed_at.isoformat()
        else:
            auto_changed_at = self.auto_changed_at

        updated_at = self.updated_at.isoformat()

        group: dict[str, Any] | None
        if isinstance(self.group, VoiceFingerprintGroup):
            group = self.group.to_dict()
        else:
            group = self.group

        thresholds_applied = self.thresholds_applied


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "auto_enabled": auto_enabled,
            "automatic_threshold": automatic_threshold,
            "suggestion_threshold": suggestion_threshold,
            "ambiguity_margin": ambiguity_margin,
            "max_candidates": max_candidates,
            "precision_threshold": precision_threshold,
            "min_confirmations": min_confirmations,
            "auto_changed_at": auto_changed_at,
            "updated_at": updated_at,
            "group": group,
            "thresholds_applied": thresholds_applied,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_fingerprint_group import VoiceFingerprintGroup # noqa: PLC0415
        d = dict(src_dict)
        auto_enabled = d.pop("auto_enabled")

        automatic_threshold = d.pop("automatic_threshold")

        suggestion_threshold = d.pop("suggestion_threshold")

        ambiguity_margin = d.pop("ambiguity_margin")

        max_candidates = d.pop("max_candidates")

        precision_threshold = d.pop("precision_threshold")

        min_confirmations = d.pop("min_confirmations")

        def _parse_auto_changed_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                auto_changed_at_type_0 = datetime.datetime.fromisoformat(data)



                return auto_changed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        auto_changed_at = _parse_auto_changed_at(d.pop("auto_changed_at"))


        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        def _parse_group(data: object) -> None | VoiceFingerprintGroup:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                group_type_1 = VoiceFingerprintGroup.from_dict(data)



                return group_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | VoiceFingerprintGroup, data)

        group = _parse_group(d.pop("group"))


        thresholds_applied = d.pop("thresholds_applied")

        voice_matching_settings = cls(
            auto_enabled=auto_enabled,
            automatic_threshold=automatic_threshold,
            suggestion_threshold=suggestion_threshold,
            ambiguity_margin=ambiguity_margin,
            max_candidates=max_candidates,
            precision_threshold=precision_threshold,
            min_confirmations=min_confirmations,
            auto_changed_at=auto_changed_at,
            updated_at=updated_at,
            group=group,
            thresholds_applied=thresholds_applied,
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
