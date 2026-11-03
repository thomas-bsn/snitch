from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PutWorkerUpdateTargetResponse200")



@_attrs_define
class PutWorkerUpdateTargetResponse200:
    """
        Attributes:
            target_version (None | str):
     """

    target_version: None | str





    def to_dict(self) -> dict[str, Any]:
        target_version: None | str
        target_version = self.target_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "target_version": target_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        def _parse_target_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        target_version = _parse_target_version(d.pop("target_version"))


        put_worker_update_target_response_200 = cls(
            target_version=target_version,
        )

        return put_worker_update_target_response_200
