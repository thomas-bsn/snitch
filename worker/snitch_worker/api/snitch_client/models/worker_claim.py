from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.job import Job





T = TypeVar("T", bound="WorkerClaim")



@_attrs_define
class WorkerClaim:
    """
        Attributes:
            job (Job | None):
            lease_token (str):
     """

    job: Job | None
    lease_token: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job import Job # noqa: PLC0415
        job: dict[str, Any] | None
        if isinstance(self.job, Job):
            job = self.job.to_dict()
        else:
            job = self.job

        lease_token = self.lease_token


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job": job,
            "lease_token": lease_token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job import Job # noqa: PLC0415
        d = dict(src_dict)
        def _parse_job(data: object) -> Job | None:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                job_type_1 = Job.from_dict(data)



                return job_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Job | None, data)

        job = _parse_job(d.pop("job"))


        lease_token = d.pop("lease_token")

        worker_claim = cls(
            job=job,
            lease_token=lease_token,
        )


        worker_claim.additional_properties = d
        return worker_claim

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
