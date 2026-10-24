from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transcription_pipeline import TranscriptionPipeline





T = TypeVar("T", bound="TranscriptionPipelineState")



@_attrs_define
class TranscriptionPipelineState:
    """
        Attributes:
            version (str):
            config (TranscriptionPipeline):
     """

    version: str
    config: TranscriptionPipeline
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
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
        from ..models.transcription_pipeline import TranscriptionPipeline # noqa: PLC0415
        d = dict(src_dict)
        version = d.pop("version")

        config = TranscriptionPipeline.from_dict(d.pop("config"))




        transcription_pipeline_state = cls(
            version=version,
            config=config,
        )


        transcription_pipeline_state.additional_properties = d
        return transcription_pipeline_state

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
