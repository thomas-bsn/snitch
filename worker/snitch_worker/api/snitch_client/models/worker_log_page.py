from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_log_line import WorkerLogLine





T = TypeVar("T", bound="WorkerLogPage")



@_attrs_define
class WorkerLogPage:
    """
        Attributes:
            session_id (str): Vide tant qu'aucune ligne n'a été reçue.
            lines (list[WorkerLogLine]):
            has_more (bool): Des lignes plus anciennes existent dans cette session.
     """

    session_id: str
    lines: list[WorkerLogLine]
    has_more: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_log_line import WorkerLogLine # noqa: PLC0415
        session_id = self.session_id

        lines = []
        for lines_item_data in self.lines:
            lines_item = lines_item_data.to_dict()
            lines.append(lines_item)



        has_more = self.has_more


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "session_id": session_id,
            "lines": lines,
            "has_more": has_more,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_log_line import WorkerLogLine # noqa: PLC0415
        d = dict(src_dict)
        session_id = d.pop("session_id")

        lines = []
        _lines = d.pop("lines")
        for lines_item_data in (_lines):
            lines_item = WorkerLogLine.from_dict(lines_item_data)



            lines.append(lines_item)


        has_more = d.pop("has_more")

        worker_log_page = cls(
            session_id=session_id,
            lines=lines,
            has_more=has_more,
        )

        return worker_log_page
