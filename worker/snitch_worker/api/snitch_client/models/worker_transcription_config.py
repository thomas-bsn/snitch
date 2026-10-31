from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerTranscriptionConfig")



@_attrs_define
class WorkerTranscriptionConfig:
    """
        Attributes:
            mode (str):
            endpoint (str):
            model (str):
            language (str):
            version (str):
            test_requested (bool | Unset):
     """

    mode: str
    endpoint: str
    model: str
    language: str
    version: str
    test_requested: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode

        endpoint = self.endpoint

        model = self.model

        language = self.language

        version = self.version

        test_requested = self.test_requested


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "endpoint": endpoint,
            "model": model,
            "language": language,
            "version": version,
        })
        if test_requested is not UNSET:
            field_dict["test_requested"] = test_requested

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = d.pop("mode")

        endpoint = d.pop("endpoint")

        model = d.pop("model")

        language = d.pop("language")

        version = d.pop("version")

        test_requested = d.pop("test_requested", UNSET)

        worker_transcription_config = cls(
            mode=mode,
            endpoint=endpoint,
            model=model,
            language=language,
            version=version,
            test_requested=test_requested,
        )


        worker_transcription_config.additional_properties = d
        return worker_transcription_config

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
