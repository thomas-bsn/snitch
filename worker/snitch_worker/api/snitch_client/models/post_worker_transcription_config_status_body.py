from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.post_worker_transcription_config_status_body_error_code import PostWorkerTranscriptionConfigStatusBodyErrorCode
from ..types import UNSET, Unset






T = TypeVar("T", bound="PostWorkerTranscriptionConfigStatusBody")



@_attrs_define
class PostWorkerTranscriptionConfigStatusBody:
    """
        Attributes:
            version (str):
            active (bool):
            error_code (PostWorkerTranscriptionConfigStatusBodyErrorCode | Unset):
     """

    version: str
    active: bool
    error_code: PostWorkerTranscriptionConfigStatusBodyErrorCode | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        active = self.active

        error_code: str | Unset = UNSET
        if not isinstance(self.error_code, Unset):
            error_code = self.error_code.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
            "active": active,
        })
        if error_code is not UNSET:
            field_dict["error_code"] = error_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        active = d.pop("active")

        _error_code = d.pop("error_code", UNSET)
        error_code: PostWorkerTranscriptionConfigStatusBodyErrorCode | Unset
        if isinstance(_error_code,  Unset):
            error_code = UNSET
        else:
            error_code = PostWorkerTranscriptionConfigStatusBodyErrorCode(_error_code)




        post_worker_transcription_config_status_body = cls(
            version=version,
            active=active,
            error_code=error_code,
        )

        return post_worker_transcription_config_status_body
