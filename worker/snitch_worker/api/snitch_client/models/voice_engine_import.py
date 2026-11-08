from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceEngineImport")



@_attrs_define
class VoiceEngineImport:
    """
        Attributes:
            key_id (str):
            payload (str):
            signature (str):
     """

    key_id: str
    payload: str
    signature: str





    def to_dict(self) -> dict[str, Any]:
        key_id = self.key_id

        payload = self.payload

        signature = self.signature


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key_id": key_id,
            "payload": payload,
            "signature": signature,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key_id = d.pop("key_id")

        payload = d.pop("payload")

        signature = d.pop("signature")

        voice_engine_import = cls(
            key_id=key_id,
            payload=payload,
            signature=signature,
        )

        return voice_engine_import
