from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_disposition_input_disposition import VoiceDispositionInputDisposition






T = TypeVar("T", bound="VoiceDispositionInput")



@_attrs_define
class VoiceDispositionInput:
    """
        Attributes:
            disposition (VoiceDispositionInputDisposition):
     """

    disposition: VoiceDispositionInputDisposition





    def to_dict(self) -> dict[str, Any]:
        disposition = self.disposition.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "disposition": disposition,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disposition = VoiceDispositionInputDisposition(d.pop("disposition"))




        voice_disposition_input = cls(
            disposition=disposition,
        )

        return voice_disposition_input
