from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WorkerLogLine")



@_attrs_define
class WorkerLogLine:
    """
        Attributes:
            id (int):
            seq (int):
            at (datetime.datetime):
            line (str):
     """

    id: int
    seq: int
    at: datetime.datetime
    line: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        seq = self.seq

        at = self.at.isoformat()

        line = self.line


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "seq": seq,
            "at": at,
            "line": line,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        seq = d.pop("seq")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        line = d.pop("line")

        worker_log_line = cls(
            id=id,
            seq=seq,
            at=at,
            line=line,
        )

        return worker_log_line
