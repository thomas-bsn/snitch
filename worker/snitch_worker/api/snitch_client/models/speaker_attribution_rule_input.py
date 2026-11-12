from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.speaker_attribution_rule_input_rule import SpeakerAttributionRuleInputRule






T = TypeVar("T", bound="SpeakerAttributionRuleInput")



@_attrs_define
class SpeakerAttributionRuleInput:
    """
        Attributes:
            rule (SpeakerAttributionRuleInputRule):
     """

    rule: SpeakerAttributionRuleInputRule





    def to_dict(self) -> dict[str, Any]:
        rule = self.rule.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "rule": rule,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        rule = SpeakerAttributionRuleInputRule(d.pop("rule"))




        speaker_attribution_rule_input = cls(
            rule=rule,
        )

        return speaker_attribution_rule_input
