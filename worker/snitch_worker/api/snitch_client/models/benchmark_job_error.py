from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BenchmarkJobError")



@_attrs_define
class BenchmarkJobError:
    """
        Attributes:
            job_id (int):
            model_id (str):
            status (str):
            error (str):
     """

    job_id: int
    model_id: str
    status: str
    error: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        model_id = self.model_id

        status = self.status

        error = self.error


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "model_id": model_id,
            "status": status,
            "error": error,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        model_id = d.pop("model_id")

        status = d.pop("status")

        error = d.pop("error")

        benchmark_job_error = cls(
            job_id=job_id,
            model_id=model_id,
            status=status,
            error=error,
        )


        benchmark_job_error.additional_properties = d
        return benchmark_job_error

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
