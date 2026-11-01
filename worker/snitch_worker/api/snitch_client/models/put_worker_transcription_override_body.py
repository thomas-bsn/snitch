from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PutWorkerTranscriptionOverrideBody")



@_attrs_define
class PutWorkerTranscriptionOverrideBody:
    """
        Attributes:
            endpoint_override (str):
     """

    endpoint_override: str





    def to_dict(self) -> dict[str, Any]:
        endpoint_override = self.endpoint_override


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "endpoint_override": endpoint_override,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        endpoint_override = d.pop("endpoint_override")

        put_worker_transcription_override_body = cls(
            endpoint_override=endpoint_override,
        )

        return put_worker_transcription_override_body
