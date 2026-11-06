from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceTraceDiarizationType0")



@_attrs_define
class VoiceTraceDiarizationType0:
    """ Null sans diarisation réussie pour cette piste.

        Attributes:
            job_id (int):
            provider (str):
            model (str):
            model_version (str):
     """

    job_id: int
    provider: str
    model: str
    model_version: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        provider = self.provider

        model = self.model

        model_version = self.model_version


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "provider": provider,
            "model": model,
            "model_version": model_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        provider = d.pop("provider")

        model = d.pop("model")

        model_version = d.pop("model_version")

        voice_trace_diarization_type_0 = cls(
            job_id=job_id,
            provider=provider,
            model=model,
            model_version=model_version,
        )


        voice_trace_diarization_type_0.additional_properties = d
        return voice_trace_diarization_type_0

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
