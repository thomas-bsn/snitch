from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="GetWorkerUpdateResponse200")



@_attrs_define
class GetWorkerUpdateResponse200:
    """
        Attributes:
            idle (bool):
            target_version (None | str):
            target_image (None | str):
     """

    idle: bool
    target_version: None | str
    target_image: None | str





    def to_dict(self) -> dict[str, Any]:
        idle = self.idle

        target_version: None | str
        target_version = self.target_version

        target_image: None | str
        target_image = self.target_image


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "idle": idle,
            "target_version": target_version,
            "target_image": target_image,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        idle = d.pop("idle")

        def _parse_target_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        target_version = _parse_target_version(d.pop("target_version"))


        def _parse_target_image(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        target_image = _parse_target_image(d.pop("target_image"))


        get_worker_update_response_200 = cls(
            idle=idle,
            target_version=target_version,
            target_image=target_image,
        )

        return get_worker_update_response_200
