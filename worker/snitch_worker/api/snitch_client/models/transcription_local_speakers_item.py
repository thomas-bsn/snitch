from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TranscriptionLocalSpeakersItem")



@_attrs_define
class TranscriptionLocalSpeakersItem:
    """
        Attributes:
            id (int):
            label (str):
            audio_stream_index (int | Unset): Piste de la voix ; présent pour une révision de Plan (une liste pour toutes
                les pistes retenues).
     """

    id: int
    label: str
    audio_stream_index: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        audio_stream_index = self.audio_stream_index


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "label": label,
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        audio_stream_index = d.pop("audio_stream_index", UNSET)

        transcription_local_speakers_item = cls(
            id=id,
            label=label,
            audio_stream_index=audio_stream_index,
        )


        transcription_local_speakers_item.additional_properties = d
        return transcription_local_speakers_item

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
