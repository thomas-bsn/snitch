from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_engine_release import VoiceEngineRelease





T = TypeVar("T", bound="VoiceEngineRegistry")



@_attrs_define
class VoiceEngineRegistry:
    """
        Attributes:
            engines (list[VoiceEngineRelease]):
     """

    engines: list[VoiceEngineRelease]





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_engine_release import VoiceEngineRelease # noqa: PLC0415
        engines = []
        for engines_item_data in self.engines:
            engines_item = engines_item_data.to_dict()
            engines.append(engines_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "engines": engines,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_engine_release import VoiceEngineRelease # noqa: PLC0415
        d = dict(src_dict)
        engines = []
        _engines = d.pop("engines")
        for engines_item_data in (_engines):
            engines_item = VoiceEngineRelease.from_dict(engines_item_data)



            engines.append(engines_item)


        voice_engine_registry = cls(
            engines=engines,
        )

        return voice_engine_registry
