from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_plan_voice_confirmation_error import AudioPlanVoiceConfirmationError
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_voice_stake import AudioPlanVoiceStake





T = TypeVar("T", bound="AudioPlanVoiceConfirmation")



@_attrs_define
class AudioPlanVoiceConfirmation:
    """
        Attributes:
            error (AudioPlanVoiceConfirmationError):
            stake (AudioPlanVoiceStake): Travail humain qu'une diarisation des pistes concernées rendrait obsolète. Les
                anciennes voix restent protégées jusqu'à un nouveau résultat réussi.
     """

    error: AudioPlanVoiceConfirmationError
    stake: AudioPlanVoiceStake
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        error = self.error.value

        stake = self.stake.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "error": error,
            "stake": stake,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_voice_stake import AudioPlanVoiceStake # noqa: PLC0415
        d = dict(src_dict)
        error = AudioPlanVoiceConfirmationError(d.pop("error"))




        stake = AudioPlanVoiceStake.from_dict(d.pop("stake"))




        audio_plan_voice_confirmation = cls(
            error=error,
            stake=stake,
        )


        audio_plan_voice_confirmation.additional_properties = d
        return audio_plan_voice_confirmation

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
