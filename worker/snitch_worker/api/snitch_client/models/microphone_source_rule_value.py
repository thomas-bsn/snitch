from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="MicrophoneSourceRuleValue")



@_attrs_define
class MicrophoneSourceRuleValue:
    """
        Attributes:
            audio_stream_index (int | None):
            title (None | str):
            person (PersonRef):
            version (int):
            updated_at (datetime.datetime):
     """

    audio_stream_index: int | None
    title: None | str
    person: PersonRef
    version: int
    updated_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        audio_stream_index: int | None
        audio_stream_index = self.audio_stream_index

        title: None | str
        title = self.title

        person = self.person.to_dict()

        version = self.version

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "audio_stream_index": audio_stream_index,
            "title": title,
            "person": person,
            "version": version,
            "updated_at": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        def _parse_audio_stream_index(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        audio_stream_index = _parse_audio_stream_index(d.pop("audio_stream_index"))


        def _parse_title(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        title = _parse_title(d.pop("title"))


        person = PersonRef.from_dict(d.pop("person"))




        version = d.pop("version")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        microphone_source_rule_value = cls(
            audio_stream_index=audio_stream_index,
            title=title,
            person=person,
            version=version,
            updated_at=updated_at,
        )


        microphone_source_rule_value.additional_properties = d
        return microphone_source_rule_value

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
