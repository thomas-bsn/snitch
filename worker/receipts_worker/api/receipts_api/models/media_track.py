from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="MediaTrack")



@_attrs_define
class MediaTrack:
    """
        Attributes:
            index (int):
            type_ (str):
            codec (str):
            title (str | Unset):
            language (str | Unset):
            width (int | Unset):
            height (int | Unset):
     """

    index: int
    type_: str
    codec: str
    title: str | Unset = UNSET
    language: str | Unset = UNSET
    width: int | Unset = UNSET
    height: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        type_ = self.type_

        codec = self.codec

        title = self.title

        language = self.language

        width = self.width

        height = self.height


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "index": index,
            "type": type_,
            "codec": codec,
        })
        if title is not UNSET:
            field_dict["title"] = title
        if language is not UNSET:
            field_dict["language"] = language
        if width is not UNSET:
            field_dict["width"] = width
        if height is not UNSET:
            field_dict["height"] = height

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        type_ = d.pop("type")

        codec = d.pop("codec")

        title = d.pop("title", UNSET)

        language = d.pop("language", UNSET)

        width = d.pop("width", UNSET)

        height = d.pop("height", UNSET)

        media_track = cls(
            index=index,
            type_=type_,
            codec=codec,
            title=title,
            language=language,
            width=width,
            height=height,
        )


        media_track.additional_properties = d
        return media_track

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
