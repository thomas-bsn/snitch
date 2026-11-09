from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkerLogBatchLinesItem")



@_attrs_define
class WorkerLogBatchLinesItem:
    """
        Attributes:
            seq (int): Numéro de la ligne dans la session, à partir de 0.
            line (str):
     """

    seq: int
    line: str





    def to_dict(self) -> dict[str, Any]:
        seq = self.seq

        line = self.line


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "seq": seq,
            "line": line,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        seq = d.pop("seq")

        line = d.pop("line")

        worker_log_batch_lines_item = cls(
            seq=seq,
            line=line,
        )

        return worker_log_batch_lines_item
