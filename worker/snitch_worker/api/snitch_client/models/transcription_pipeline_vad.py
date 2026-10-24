from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcription_pipeline_vad_mode import TranscriptionPipelineVadMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="TranscriptionPipelineVad")



@_attrs_define
class TranscriptionPipelineVad:
    """
        Attributes:
            mode (TranscriptionPipelineVadMode):
            version (str):
            threshold (float | Unset):
            min_speech_ms (int | Unset):
            min_silence_ms (int | Unset):
            padding_ms (int | Unset):
     """

    mode: TranscriptionPipelineVadMode
    version: str
    threshold: float | Unset = UNSET
    min_speech_ms: int | Unset = UNSET
    min_silence_ms: int | Unset = UNSET
    padding_ms: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        version = self.version

        threshold = self.threshold

        min_speech_ms = self.min_speech_ms

        min_silence_ms = self.min_silence_ms

        padding_ms = self.padding_ms


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "version": version,
        })
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if min_speech_ms is not UNSET:
            field_dict["min_speech_ms"] = min_speech_ms
        if min_silence_ms is not UNSET:
            field_dict["min_silence_ms"] = min_silence_ms
        if padding_ms is not UNSET:
            field_dict["padding_ms"] = padding_ms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = TranscriptionPipelineVadMode(d.pop("mode"))




        version = d.pop("version")

        threshold = d.pop("threshold", UNSET)

        min_speech_ms = d.pop("min_speech_ms", UNSET)

        min_silence_ms = d.pop("min_silence_ms", UNSET)

        padding_ms = d.pop("padding_ms", UNSET)

        transcription_pipeline_vad = cls(
            mode=mode,
            version=version,
            threshold=threshold,
            min_speech_ms=min_speech_ms,
            min_silence_ms=min_silence_ms,
            padding_ms=padding_ms,
        )


        transcription_pipeline_vad.additional_properties = d
        return transcription_pipeline_vad

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
