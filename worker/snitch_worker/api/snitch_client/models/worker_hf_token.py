from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WorkerHFToken")



@_attrs_define
class WorkerHFToken:
    """
        Attributes:
            version (str):
            token (None | str): null : copie centrale à purger
     """

    version: str
    token: None | str





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        token: None | str
        token = self.token


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
            "token": token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        def _parse_token(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        token = _parse_token(d.pop("token"))


        worker_hf_token = cls(
            version=version,
            token=token,
        )

        return worker_hf_token
