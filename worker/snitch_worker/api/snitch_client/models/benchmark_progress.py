from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.benchmark_job_error import BenchmarkJobError





T = TypeVar("T", bound="BenchmarkProgress")



@_attrs_define
class BenchmarkProgress:
    """
        Attributes:
            jobs (int):
            done (int):
            failed (int):
            active (int):
            errors (list[BenchmarkJobError]):
     """

    jobs: int
    done: int
    failed: int
    active: int
    errors: list[BenchmarkJobError]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.benchmark_job_error import BenchmarkJobError # noqa: PLC0415
        jobs = self.jobs

        done = self.done

        failed = self.failed

        active = self.active

        errors = []
        for errors_item_data in self.errors:
            errors_item = errors_item_data.to_dict()
            errors.append(errors_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "jobs": jobs,
            "done": done,
            "failed": failed,
            "active": active,
            "errors": errors,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.benchmark_job_error import BenchmarkJobError # noqa: PLC0415
        d = dict(src_dict)
        jobs = d.pop("jobs")

        done = d.pop("done")

        failed = d.pop("failed")

        active = d.pop("active")

        errors = []
        _errors = d.pop("errors")
        for errors_item_data in (_errors):
            errors_item = BenchmarkJobError.from_dict(errors_item_data)



            errors.append(errors_item)


        benchmark_progress = cls(
            jobs=jobs,
            done=done,
            failed=failed,
            active=active,
            errors=errors,
        )


        benchmark_progress.additional_properties = d
        return benchmark_progress

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
