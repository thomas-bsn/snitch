from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceModelPatch")



@_attrs_define
class VoiceModelPatch:
    """
        Attributes:
            label (str | Unset):
            license_ (str | Unset):
            size_bytes (int | Unset):
            vram_bytes (int | Unset):
            enabled (bool | Unset):
            family (str | Unset): Famille d'engine à la demande du modèle (ADR 0056) ; chaîne vide pour revenir au contrat
                embarqué ou signé. Doit exister dans la table des familles et couvrir la tâche du modèle (400
                unsupported_engine_family) ; les modèles à adaptateur embarqué (pyannote, silero-jit) n'en reçoivent pas. Refusé
                (409 voice_model_order_in_progress) tant qu'un ordre ou une installation du modèle est ouvert.
     """

    label: str | Unset = UNSET
    license_: str | Unset = UNSET
    size_bytes: int | Unset = UNSET
    vram_bytes: int | Unset = UNSET
    enabled: bool | Unset = UNSET
    family: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        license_ = self.license_

        size_bytes = self.size_bytes

        vram_bytes = self.vram_bytes

        enabled = self.enabled

        family = self.family


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if label is not UNSET:
            field_dict["label"] = label
        if license_ is not UNSET:
            field_dict["license"] = license_
        if size_bytes is not UNSET:
            field_dict["size_bytes"] = size_bytes
        if vram_bytes is not UNSET:
            field_dict["vram_bytes"] = vram_bytes
        if enabled is not UNSET:
            field_dict["enabled"] = enabled
        if family is not UNSET:
            field_dict["family"] = family

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label", UNSET)

        license_ = d.pop("license", UNSET)

        size_bytes = d.pop("size_bytes", UNSET)

        vram_bytes = d.pop("vram_bytes", UNSET)

        enabled = d.pop("enabled", UNSET)

        family = d.pop("family", UNSET)

        voice_model_patch = cls(
            label=label,
            license_=license_,
            size_bytes=size_bytes,
            vram_bytes=vram_bytes,
            enabled=enabled,
            family=family,
        )

        return voice_model_patch
