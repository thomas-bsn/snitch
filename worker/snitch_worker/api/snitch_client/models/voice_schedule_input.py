from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceScheduleInput")



@_attrs_define
class VoiceScheduleInput:
    """
        Attributes:
            audio_stream_index (int | Unset):
            confirm_replace_validations (bool | Unset): Vrai pour confirmer explicitement la relance sur un clip qui
                contient des validations humaines (locuteurs validés, échantillons vocaux) ou des corrections de locuteur. Sans
                lui, la relance est refusée (409 voice_rediarization_confirmation_required) avec ce qui est en jeu.
     """

    audio_stream_index: int | Unset = UNSET
    confirm_replace_validations: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        audio_stream_index = self.audio_stream_index

        confirm_replace_validations = self.confirm_replace_validations


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
        if confirm_replace_validations is not UNSET:
            field_dict["confirm_replace_validations"] = confirm_replace_validations

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audio_stream_index = d.pop("audio_stream_index", UNSET)

        confirm_replace_validations = d.pop("confirm_replace_validations", UNSET)

        voice_schedule_input = cls(
            audio_stream_index=audio_stream_index,
            confirm_replace_validations=confirm_replace_validations,
        )


        voice_schedule_input.additional_properties = d
        return voice_schedule_input

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
