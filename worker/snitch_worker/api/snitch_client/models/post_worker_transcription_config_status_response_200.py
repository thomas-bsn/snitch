from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PostWorkerTranscriptionConfigStatusResponse200")



@_attrs_define
class PostWorkerTranscriptionConfigStatusResponse200:
    """
        Attributes:
            received_version (str):
            tested_version (str):
            active_version (str):
            error_code (str):
     """

    received_version: str
    tested_version: str
    active_version: str
    error_code: str





    def to_dict(self) -> dict[str, Any]:
        received_version = self.received_version

        tested_version = self.tested_version

        active_version = self.active_version

        error_code = self.error_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "received_version": received_version,
            "tested_version": tested_version,
            "active_version": active_version,
            "error_code": error_code,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        received_version = d.pop("received_version")

        tested_version = d.pop("tested_version")

        active_version = d.pop("active_version")

        error_code = d.pop("error_code")

        post_worker_transcription_config_status_response_200 = cls(
            received_version=received_version,
            tested_version=tested_version,
            active_version=active_version,
            error_code=error_code,
        )

        return post_worker_transcription_config_status_response_200
