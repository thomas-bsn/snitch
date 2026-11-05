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
     """

    label: str | Unset = UNSET
    license_: str | Unset = UNSET
    size_bytes: int | Unset = UNSET
    vram_bytes: int | Unset = UNSET
    enabled: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        license_ = self.license_

        size_bytes = self.size_bytes

        vram_bytes = self.vram_bytes

        enabled = self.enabled


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

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label", UNSET)

        license_ = d.pop("license", UNSET)

        size_bytes = d.pop("size_bytes", UNSET)

        vram_bytes = d.pop("vram_bytes", UNSET)

        enabled = d.pop("enabled", UNSET)

        voice_model_patch = cls(
            label=label,
            license_=license_,
            size_bytes=size_bytes,
            vram_bytes=vram_bytes,
            enabled=enabled,
        )

        return voice_model_patch
