from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.worker_result_input_result import WorkerResultInputResult





T = TypeVar("T", bound="WorkerResultInput")



@_attrs_define
class WorkerResultInput:
    """
        Attributes:
            input_version (str):
            result (WorkerResultInputResult):
     """

    input_version: str
    result: WorkerResultInputResult





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_result_input_result import WorkerResultInputResult # noqa: PLC0415
        input_version = self.input_version

        result = self.result.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "input_version": input_version,
            "result": result,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_result_input_result import WorkerResultInputResult # noqa: PLC0415
        d = dict(src_dict)
        input_version = d.pop("input_version")

        result = WorkerResultInputResult.from_dict(d.pop("result"))




        worker_result_input = cls(
            input_version=input_version,
            result=result,
        )

        return worker_result_input
