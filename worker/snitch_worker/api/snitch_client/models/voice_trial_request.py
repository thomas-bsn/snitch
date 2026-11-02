from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VoiceTrialRequest")



@_attrs_define
class VoiceTrialRequest:
    """
        Attributes:
            model_id (str):
            audio_stream_index (int | Unset):
            num_speakers (int | Unset):
            min_speakers (int | Unset):
            max_speakers (int | Unset):
     """

    model_id: str
    audio_stream_index: int | Unset = UNSET
    num_speakers: int | Unset = UNSET
    min_speakers: int | Unset = UNSET
    max_speakers: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        model_id = self.model_id

        audio_stream_index = self.audio_stream_index

        num_speakers = self.num_speakers

        min_speakers = self.min_speakers

        max_speakers = self.max_speakers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model_id": model_id,
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
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
        model_id = d.pop("model_id")

        audio_stream_index = d.pop("audio_stream_index", UNSET)

        num_speakers = d.pop("num_speakers", UNSET)

        min_speakers = d.pop("min_speakers", UNSET)

        max_speakers = d.pop("max_speakers", UNSET)

        voice_trial_request = cls(
            model_id=model_id,
            audio_stream_index=audio_stream_index,
            num_speakers=num_speakers,
            min_speakers=min_speakers,
            max_speakers=max_speakers,
        )

        return voice_trial_request
