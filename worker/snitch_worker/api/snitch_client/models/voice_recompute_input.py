from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceRecomputeInput")



@_attrs_define
class VoiceRecomputeInput:
    """
        Attributes:
            confirm_fingerprints (int): Nombre d'empreintes vu dans l'aperçu.
     """

    confirm_fingerprints: int





    def to_dict(self) -> dict[str, Any]:
        confirm_fingerprints = self.confirm_fingerprints


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirm_fingerprints": confirm_fingerprints,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirm_fingerprints = d.pop("confirm_fingerprints")

        voice_recompute_input = cls(
            confirm_fingerprints=confirm_fingerprints,
        )

        return voice_recompute_input
