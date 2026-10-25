from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceAccuracyCriteria")



@_attrs_define
class VoiceAccuracyCriteria:
    """
        Attributes:
            met (bool):
            precision_met (bool):
            confirmations_met (bool):
     """

    met: bool
    precision_met: bool
    confirmations_met: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        met = self.met

        precision_met = self.precision_met

        confirmations_met = self.confirmations_met


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "met": met,
            "precision_met": precision_met,
            "confirmations_met": confirmations_met,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        met = d.pop("met")

        precision_met = d.pop("precision_met")

        confirmations_met = d.pop("confirmations_met")

        voice_accuracy_criteria = cls(
            met=met,
            precision_met=precision_met,
            confirmations_met=confirmations_met,
        )


        voice_accuracy_criteria.additional_properties = d
        return voice_accuracy_criteria

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
