from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_log_batch_lines_item import WorkerLogBatchLinesItem





T = TypeVar("T", bound="WorkerLogBatch")



@_attrs_define
class WorkerLogBatch:
    """
        Attributes:
            session_id (str): Un processus du Worker.
            lines (list[WorkerLogBatchLinesItem]):
     """

    session_id: str
    lines: list[WorkerLogBatchLinesItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_log_batch_lines_item import WorkerLogBatchLinesItem # noqa: PLC0415
        session_id = self.session_id

        lines = []
        for lines_item_data in self.lines:
            lines_item = lines_item_data.to_dict()
            lines.append(lines_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "session_id": session_id,
            "lines": lines,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_log_batch_lines_item import WorkerLogBatchLinesItem # noqa: PLC0415
        d = dict(src_dict)
        session_id = d.pop("session_id")

        lines = []
        _lines = d.pop("lines")
        for lines_item_data in (_lines):
            lines_item = WorkerLogBatchLinesItem.from_dict(lines_item_data)



            lines.append(lines_item)


        worker_log_batch = cls(
            session_id=session_id,
            lines=lines,
        )

        return worker_log_batch
