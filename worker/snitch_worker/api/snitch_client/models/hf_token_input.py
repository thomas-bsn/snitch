from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HFTokenInput")



@_attrs_define
class HFTokenInput:
    """
        Attributes:
            token (str): Token Hugging Face : écrit une fois, jamais renvoyé
     """

    token: str





    def to_dict(self) -> dict[str, Any]:
        token = self.token


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "token": token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        token = d.pop("token")

        hf_token_input = cls(
            token=token,
        )

        return hf_token_input
