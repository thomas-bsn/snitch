from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.get_worker_transcription_config_response_200_mode import GetWorkerTranscriptionConfigResponse200Mode






T = TypeVar("T", bound="GetWorkerTranscriptionConfigResponse200")



@_attrs_define
class GetWorkerTranscriptionConfigResponse200:
    """
        Attributes:
            mode (GetWorkerTranscriptionConfigResponse200Mode):
            endpoint (str):
            model (str):
            language (str):
            version (str):
            test_requested (bool):
            endpoint_override (str):
     """

    mode: GetWorkerTranscriptionConfigResponse200Mode
    endpoint: str
    model: str
    language: str
    version: str
    test_requested: bool
    endpoint_override: str





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        endpoint = self.endpoint

        model = self.model

        language = self.language

        version = self.version

        test_requested = self.test_requested

        endpoint_override = self.endpoint_override


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "endpoint": endpoint,
            "model": model,
            "language": language,
            "version": version,
            "test_requested": test_requested,
            "endpoint_override": endpoint_override,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = GetWorkerTranscriptionConfigResponse200Mode(d.pop("mode"))




        endpoint = d.pop("endpoint")

        model = d.pop("model")

        language = d.pop("language")

        version = d.pop("version")

        test_requested = d.pop("test_requested")

        endpoint_override = d.pop("endpoint_override")

        get_worker_transcription_config_response_200 = cls(
            mode=mode,
            endpoint=endpoint,
            model=model,
            language=language,
            version=version,
            test_requested=test_requested,
            endpoint_override=endpoint_override,
        )

        return get_worker_transcription_config_response_200
