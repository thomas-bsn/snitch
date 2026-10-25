from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.person_ref import PersonRef





T = TypeVar("T", bound="MicrophoneSnapshot")



@_attrs_define
class MicrophoneSnapshot:
    """ Désignation telle qu'elle était, avec le nom de la Personne à l'époque.

        Attributes:
            audio_stream_index (int | Unset):
            title (str | Unset):
            disabled (bool | Unset):
            person (PersonRef | Unset):
     """

    audio_stream_index: int | Unset = UNSET
    title: str | Unset = UNSET
    disabled: bool | Unset = UNSET
    person: PersonRef | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        audio_stream_index = self.audio_stream_index

        title = self.title

        disabled = self.disabled

        person: dict[str, Any] | Unset = UNSET
        if not isinstance(self.person, Unset):
            person = self.person.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
        })
        if audio_stream_index is not UNSET:
            field_dict["audio_stream_index"] = audio_stream_index
        if title is not UNSET:
            field_dict["title"] = title
        if disabled is not UNSET:
            field_dict["disabled"] = disabled
        if person is not UNSET:
            field_dict["person"] = person

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.person_ref import PersonRef # noqa: PLC0415
        d = dict(src_dict)
        audio_stream_index = d.pop("audio_stream_index", UNSET)

        title = d.pop("title", UNSET)

        disabled = d.pop("disabled", UNSET)

        _person = d.pop("person", UNSET)
        person: PersonRef | Unset
        if isinstance(_person,  Unset):
            person = UNSET
        else:
            person = PersonRef.from_dict(_person)




        microphone_snapshot = cls(
            audio_stream_index=audio_stream_index,
            title=title,
            disabled=disabled,
            person=person,
        )


        microphone_snapshot.additional_properties = d
        return microphone_snapshot

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
