from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.job_log_line import JobLogLine





T = TypeVar("T", bound="JobLogs")



@_attrs_define
class JobLogs:
    """
        Attributes:
            job_id (int):
            lines (list[JobLogLine]):
     """

    job_id: int
    lines: list[JobLogLine]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job_log_line import JobLogLine # noqa: PLC0415
        job_id = self.job_id

        lines = []
        for lines_item_data in self.lines:
            lines_item = lines_item_data.to_dict()
            lines.append(lines_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "lines": lines,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_log_line import JobLogLine # noqa: PLC0415
        d = dict(src_dict)
        job_id = d.pop("job_id")

        lines = []
        _lines = d.pop("lines")
        for lines_item_data in (_lines):
            lines_item = JobLogLine.from_dict(lines_item_data)



            lines.append(lines_item)


        job_logs = cls(
            job_id=job_id,
            lines=lines,
        )


        job_logs.additional_properties = d
        return job_logs

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
