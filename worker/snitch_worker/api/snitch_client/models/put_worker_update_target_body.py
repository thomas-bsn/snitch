from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PutWorkerUpdateTargetBody")



@_attrs_define
class PutWorkerUpdateTargetBody:
    """
        Attributes:
            version (None | str):
     """

    version: None | str





    def to_dict(self) -> dict[str, Any]:
        version: None | str
        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        def _parse_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        version = _parse_version(d.pop("version"))


        put_worker_update_target_body = cls(
            version=version,
        )

        return put_worker_update_target_body
