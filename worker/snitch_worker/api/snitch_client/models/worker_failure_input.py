from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerFailureInput")



@_attrs_define
class WorkerFailureInput:
    """
        Attributes:
            error (str):
     """

    error: str





    def to_dict(self) -> dict[str, Any]:
        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "error": error,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        error = d.pop("error")

        worker_failure_input = cls(
            error=error,
        )

        return worker_failure_input
