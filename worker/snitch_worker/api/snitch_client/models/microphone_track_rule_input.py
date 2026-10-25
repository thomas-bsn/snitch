from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="MicrophoneTrackRuleInput")



@_attrs_define
class MicrophoneTrackRuleInput:
    """ Au moins un de audio_stream_index ou title ; si les deux sont donnés, la piste doit correspondre aux deux.

        Attributes:
            person_id (int): Personne propriétaire du micro.
            audio_stream_index (int | Unset): Index global FFmpeg d'une piste détectée.
            title (str | Unset): Titre exact (sans casse) d'une piste détectée.
     """

    person_id: int
    audio_stream_index: int | Unset = UNSET
    title: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        person_id = self.person_id

        audio_stream_index = self.audio_stream_index

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "person_id": person_id,
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        person_id = d.pop("person_id")

        audio_stream_index = d.pop("audio_stream_index", UNSET)

        title = d.pop("title", UNSET)

        microphone_track_rule_input = cls(
            person_id=person_id,
            audio_stream_index=audio_stream_index,
            title=title,
        )

        return microphone_track_rule_input
