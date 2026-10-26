from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceAutomationCriteriaInput")



@_attrs_define
class VoiceAutomationCriteriaInput:
    """
        Attributes:
            precision_threshold (float):
            min_confirmations (int):
     """

    precision_threshold: float
    min_confirmations: int





    def to_dict(self) -> dict[str, Any]:
        precision_threshold = self.precision_threshold

        min_confirmations = self.min_confirmations


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "precision_threshold": precision_threshold,
            "min_confirmations": min_confirmations,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        precision_threshold = d.pop("precision_threshold")

        min_confirmations = d.pop("min_confirmations")

        voice_automation_criteria_input = cls(
            precision_threshold=precision_threshold,
            min_confirmations=min_confirmations,
        )

        return voice_automation_criteria_input
