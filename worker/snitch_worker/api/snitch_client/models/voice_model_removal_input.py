from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceModelRemovalInput")



@_attrs_define
class VoiceModelRemovalInput:
    """
        Attributes:
            model_id (str):
            idempotency_key (str):
     """

    model_id: str
    idempotency_key: str





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        idempotency_key = self.idempotency_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model_id": model_id,
            "idempotency_key": idempotency_key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model_id = d.pop("model_id")

        idempotency_key = d.pop("idempotency_key")

        voice_model_removal_input = cls(
            model_id=model_id,
            idempotency_key=idempotency_key,
        )

        return voice_model_removal_input
