from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.voice_pipeline import VoicePipeline





T = TypeVar("T", bound="VoicePipelineState")



@_attrs_define
class VoicePipelineState:
    """
        Attributes:
            version (str):
            config (VoicePipeline):
     """

    version: str
    config: VoicePipeline
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_pipeline import VoicePipeline # noqa: PLC0415
        version = self.version

        config = self.config.to_dict()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "version": version,
            "config": config,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_pipeline import VoicePipeline # noqa: PLC0415
        d = dict(src_dict)
        version = d.pop("version")

        config = VoicePipeline.from_dict(d.pop("config"))




        voice_pipeline_state = cls(
            version=version,
            config=config,
        )


        voice_pipeline_state.additional_properties = d
        return voice_pipeline_state

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
