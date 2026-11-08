from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RevokeVoiceEngineResponse200")



@_attrs_define
class RevokeVoiceEngineResponse200:
    """
        Attributes:
            revoked (bool):
     """

    revoked: bool





    def to_dict(self) -> dict[str, Any]:
        revoked = self.revoked


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "revoked": revoked,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        revoked = d.pop("revoked")

        revoke_voice_engine_response_200 = cls(
            revoked=revoked,
        )

        return revoke_voice_engine_response_200
