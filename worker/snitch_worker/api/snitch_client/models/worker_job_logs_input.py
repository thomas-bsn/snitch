from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WorkerJobLogsInput")



@_attrs_define
class WorkerJobLogsInput:
    """
        Attributes:
            lines (list[str]):
     """

    lines: list[str]





    def to_dict(self) -> dict[str, Any]:
        lines = self.lines




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lines": lines,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        lines = cast(list[str], d.pop("lines"))


        worker_job_logs_input = cls(
            lines=lines,
        )

        return worker_job_logs_input
