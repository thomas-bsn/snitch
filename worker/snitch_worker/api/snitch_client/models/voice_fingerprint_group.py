from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceFingerprintGroup")



@_attrs_define
class VoiceFingerprintGroup:
    """ Groupe d'empreintes (modèle, version, dimensions) : seules les empreintes d'un même groupe sont comparées.

        Attributes:
            model (str):
            model_version (str):
            dimensions (int):
     """

    model: str
    model_version: str
    dimensions: int





    def to_dict(self) -> dict[str, Any]:
        model = self.model

        model_version = self.model_version

        dimensions = self.dimensions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model": model,
            "model_version": model_version,
            "dimensions": dimensions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model = d.pop("model")

        model_version = d.pop("model_version")

        dimensions = d.pop("dimensions")

        voice_fingerprint_group = cls(
            model=model,
            model_version=model_version,
            dimensions=dimensions,
        )

        return voice_fingerprint_group
