from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_trial_config_provider import VoiceTrialConfigProvider
from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceTrialConfig")



@_attrs_define
class VoiceTrialConfig:
    """
        Attributes:
            provider (VoiceTrialConfigProvider):
            model (str):
            model_version (str):
            num_speakers (int | Unset):
            min_speakers (int | Unset):
            max_speakers (int | Unset):
     """

    provider: VoiceTrialConfigProvider
    model: str
    model_version: str
    num_speakers: int | Unset = UNSET
    min_speakers: int | Unset = UNSET
    max_speakers: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        provider = self.provider.value

        model = self.model

        model_version = self.model_version

        num_speakers = self.num_speakers

        min_speakers = self.min_speakers

        max_speakers = self.max_speakers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "provider": provider,
            "model": model,
            "model_version": model_version,
        })
        if num_speakers is not UNSET:
            field_dict["num_speakers"] = num_speakers
        if min_speakers is not UNSET:
            field_dict["min_speakers"] = min_speakers
        if max_speakers is not UNSET:
            field_dict["max_speakers"] = max_speakers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider = VoiceTrialConfigProvider(d.pop("provider"))




        model = d.pop("model")

        model_version = d.pop("model_version")

        num_speakers = d.pop("num_speakers", UNSET)

        min_speakers = d.pop("min_speakers", UNSET)

        max_speakers = d.pop("max_speakers", UNSET)

        voice_trial_config = cls(
            provider=provider,
            model=model,
            model_version=model_version,
            num_speakers=num_speakers,
            min_speakers=min_speakers,
            max_speakers=max_speakers,
        )

        return voice_trial_config
