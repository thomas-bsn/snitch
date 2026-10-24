from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_track_selection_status import AudioTrackSelectionStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="AudioTrackSelection")



@_attrs_define
class AudioTrackSelection:
    """
        Attributes:
            status (AudioTrackSelectionStatus):
            audio_stream_index (int | Unset):
            selection_source (str | Unset):
            rule_version (int | Unset):
     """

    status: AudioTrackSelectionStatus
    audio_stream_index: int | Unset = UNSET
    selection_source: str | Unset = UNSET
    rule_version: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        audio_stream_index = self.audio_stream_index

        selection_source = self.selection_source

        rule_version = self.rule_version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "status": status,
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
        if selection_source is not UNSET:
            field_dict["selection_source"] = selection_source
        if rule_version is not UNSET:
            field_dict["rule_version"] = rule_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = AudioTrackSelectionStatus(d.pop("status"))




        audio_stream_index = d.pop("audio_stream_index", UNSET)

        selection_source = d.pop("selection_source", UNSET)

        rule_version = d.pop("rule_version", UNSET)

        audio_track_selection = cls(
            status=status,
            audio_stream_index=audio_stream_index,
            selection_source=selection_source,
            rule_version=rule_version,
        )


        audio_track_selection.additional_properties = d
        return audio_track_selection

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
