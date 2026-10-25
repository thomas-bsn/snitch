from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="MicrophoneTrackClipInput")



@_attrs_define
class MicrophoneTrackClipInput:
    """ Soit disabled=true seul, soit audio_stream_index et person_id ensemble.

        Attributes:
            disabled (bool | Unset): true : ce clip n'a pas de piste micro.
            audio_stream_index (int | Unset):
            person_id (int | Unset):
     """

    disabled: bool | Unset = UNSET
    audio_stream_index: int | Unset = UNSET
    person_id: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        disabled = self.disabled

        audio_stream_index = self.audio_stream_index

        person_id = self.person_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if disabled is not UNSET:
            field_dict["disabled"] = disabled
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
        if person_id is not UNSET:
            field_dict["person_id"] = person_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disabled = d.pop("disabled", UNSET)

        audio_stream_index = d.pop("audio_stream_index", UNSET)

        person_id = d.pop("person_id", UNSET)

        microphone_track_clip_input = cls(
            disabled=disabled,
            audio_stream_index=audio_stream_index,
            person_id=person_id,
        )

        return microphone_track_clip_input
