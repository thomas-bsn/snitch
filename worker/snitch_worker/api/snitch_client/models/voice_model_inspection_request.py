from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceModelInspectionRequest")



@_attrs_define
class VoiceModelInspectionRequest:
    """
        Attributes:
            source (str): Dépôt Hugging Face : owner/nom ou adresse huggingface.co.
     """

    source: str





    def to_dict(self) -> dict[str, Any]:
        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = d.pop("source")

        voice_model_inspection_request = cls(
            source=source,
        )

        return voice_model_inspection_request
