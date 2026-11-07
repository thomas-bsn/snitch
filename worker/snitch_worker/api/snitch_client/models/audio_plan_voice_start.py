from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AudioPlanVoiceStart")



@_attrs_define
class AudioPlanVoiceStart:
    """
        Attributes:
            confirm_replace_validations (bool | Unset): Vrai pour confirmer le remplacement des validations, échantillons
                vocaux (Références) et corrections de locuteur des pistes recensées dans la confirmation 409.
     """

    confirm_replace_validations: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        confirm_replace_validations = self.confirm_replace_validations


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if confirm_replace_validations is not UNSET:
            field_dict["confirm_replace_validations"] = confirm_replace_validations

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirm_replace_validations = d.pop("confirm_replace_validations", UNSET)

        audio_plan_voice_start = cls(
            confirm_replace_validations=confirm_replace_validations,
        )


        audio_plan_voice_start.additional_properties = d
        return audio_plan_voice_start

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
