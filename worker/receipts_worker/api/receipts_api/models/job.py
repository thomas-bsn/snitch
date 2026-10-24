from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.job_status import JobStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.job_result_type_0 import JobResultType0





T = TypeVar("T", bound="Job")



@_attrs_define
class Job:
    """
        Attributes:
            id (int):
            clip_id (int):
            step (str):
            input_version (str):
            status (JobStatus):
            priority (int):
            attempts (int):
            max_attempts (int):
            lease_owner (None | str):
            lease_expires_at (datetime.datetime | None):
            next_attempt_at (datetime.datetime):
            last_error (str):
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            result (JobResultType0 | None | Unset):
     """

    id: int
    clip_id: int
    step: str
    input_version: str
    status: JobStatus
    priority: int
    attempts: int
    max_attempts: int
    lease_owner: None | str
    lease_expires_at: datetime.datetime | None
    next_attempt_at: datetime.datetime
    last_error: str
    created_at: datetime.datetime
    updated_at: datetime.datetime
    result: JobResultType0 | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.job_result_type_0 import JobResultType0 # noqa: PLC0415
        id = self.id

        clip_id = self.clip_id

        step = self.step

        input_version = self.input_version

        status = self.status.value

        priority = self.priority

        attempts = self.attempts

        max_attempts = self.max_attempts

        lease_owner: None | str
        lease_owner = self.lease_owner

        lease_expires_at: None | str
        if isinstance(self.lease_expires_at, datetime.datetime):
            lease_expires_at = self.lease_expires_at.isoformat()
        else:
            lease_expires_at = self.lease_expires_at

        next_attempt_at = self.next_attempt_at.isoformat()

        last_error = self.last_error

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        result: dict[str, Any] | None | Unset
        if isinstance(self.result, Unset):
            result = UNSET
        elif isinstance(self.result, JobResultType0):
            result = self.result.to_dict()
        else:
            result = self.result


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "clip_id": clip_id,
            "step": step,
            "input_version": input_version,
            "status": status,
            "priority": priority,
            "attempts": attempts,
            "max_attempts": max_attempts,
            "lease_owner": lease_owner,
            "lease_expires_at": lease_expires_at,
            "next_attempt_at": next_attempt_at,
            "last_error": last_error,
            "created_at": created_at,
            "updated_at": updated_at,
        })
        if result is not UNSET:
            field_dict["result"] = result

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_result_type_0 import JobResultType0 # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        clip_id = d.pop("clip_id")

        step = d.pop("step")

        input_version = d.pop("input_version")

        status = JobStatus(d.pop("status"))




        priority = d.pop("priority")

        attempts = d.pop("attempts")

        max_attempts = d.pop("max_attempts")

        def _parse_lease_owner(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        lease_owner = _parse_lease_owner(d.pop("lease_owner"))


        def _parse_lease_expires_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                lease_expires_at_type_0 = datetime.datetime.fromisoformat(data)



                return lease_expires_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        lease_expires_at = _parse_lease_expires_at(d.pop("lease_expires_at"))


        next_attempt_at = datetime.datetime.fromisoformat(d.pop("next_attempt_at"))




        last_error = d.pop("last_error")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        def _parse_result(data: object) -> JobResultType0 | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                result_type_0 = JobResultType0.from_dict(data)



                return result_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(JobResultType0 | None | Unset, data)

        result = _parse_result(d.pop("result", UNSET))


        job = cls(
            id=id,
            clip_id=clip_id,
            step=step,
            input_version=input_version,
            status=status,
            priority=priority,
            attempts=attempts,
            max_attempts=max_attempts,
            lease_owner=lease_owner,
            lease_expires_at=lease_expires_at,
            next_attempt_at=next_attempt_at,
            last_error=last_error,
            created_at=created_at,
            updated_at=updated_at,
            result=result,
        )


        job.additional_properties = d
        return job

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
