from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceTraceTranscription")



@_attrs_define
class VoiceTraceTranscription:
    """
        Attributes:
            job_id (int):
            provider (str):
            model (str):
            version (str):
            language (str):
            segment_count (int):
     """

    job_id: int
    provider: str
    model: str
    version: str
    language: str
    segment_count: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        provider = self.provider

        model = self.model

        version = self.version

        language = self.language

        segment_count = self.segment_count


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "provider": provider,
            "model": model,
            "version": version,
            "language": language,
            "segment_count": segment_count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        provider = d.pop("provider")

        model = d.pop("model")

        version = d.pop("version")

        language = d.pop("language")

        segment_count = d.pop("segment_count")

        voice_trace_transcription = cls(
            job_id=job_id,
            provider=provider,
            model=model,
            version=version,
            language=language,
            segment_count=segment_count,
        )


        voice_trace_transcription.additional_properties = d
        return voice_trace_transcription

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
