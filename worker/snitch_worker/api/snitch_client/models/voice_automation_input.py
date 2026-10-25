from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceAutomationInput")



@_attrs_define
class VoiceAutomationInput:
    """
        Attributes:
            enabled (bool):
            confirm_unmet_criteria (bool | Unset): Requis pour activer alors que les critères ne sont pas atteints ; sans
                effet sinon.
     """

    enabled: bool
    confirm_unmet_criteria: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        confirm_unmet_criteria = self.confirm_unmet_criteria


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
        })
        if confirm_unmet_criteria is not UNSET:
            field_dict["confirm_unmet_criteria"] = confirm_unmet_criteria

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        confirm_unmet_criteria = d.pop("confirm_unmet_criteria", UNSET)

        voice_automation_input = cls(
            enabled=enabled,
            confirm_unmet_criteria=confirm_unmet_criteria,
        )

        return voice_automation_input
