from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="FingerprintTransitionInput")



@_attrs_define
class FingerprintTransitionInput:
    """
        Attributes:
            to_model_id (str):
            confirm_references (int):
     """

    to_model_id: str
    confirm_references: int





    def to_dict(self) -> dict[str, Any]:
        to_model_id = self.to_model_id

        confirm_references = self.confirm_references


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "to_model_id": to_model_id,
            "confirm_references": confirm_references,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        to_model_id = d.pop("to_model_id")

        confirm_references = d.pop("confirm_references")

        fingerprint_transition_input = cls(
            to_model_id=to_model_id,
            confirm_references=confirm_references,
        )

        return fingerprint_transition_input
