from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceEngineWheel")



@_attrs_define
class VoiceEngineWheel:
    """
        Attributes:
            name (str):
            url (str):
            sha256 (str):
            size (int):
     """

    name: str
    url: str
    sha256: str
    size: int





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        url = self.url

        sha256 = self.sha256

        size = self.size


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "url": url,
            "sha256": sha256,
            "size": size,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        url = d.pop("url")

        sha256 = d.pop("sha256")

        size = d.pop("size")

        voice_engine_wheel = cls(
            name=name,
            url=url,
            sha256=sha256,
            size=size,
        )

        return voice_engine_wheel
