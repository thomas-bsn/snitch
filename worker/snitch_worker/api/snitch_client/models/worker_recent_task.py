from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_recent_task_kind import WorkerRecentTaskKind
from ..models.worker_recent_task_outcome import WorkerRecentTaskOutcome
from typing import cast
import datetime






T = TypeVar("T", bound="WorkerRecentTask")



@_attrs_define
class WorkerRecentTask:
    """
        Attributes:
            job_id (int):
            clip_id (int):
            kind (WorkerRecentTaskKind):
            outcome (WorkerRecentTaskOutcome):
            error_code (str):
            at (datetime.datetime):
     """

    job_id: int
    clip_id: int
    kind: WorkerRecentTaskKind
    outcome: WorkerRecentTaskOutcome
    error_code: str
    at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        clip_id = self.clip_id

        kind = self.kind.value

        outcome = self.outcome.value

        error_code = self.error_code

        at = self.at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "clip_id": clip_id,
            "kind": kind,
            "outcome": outcome,
            "error_code": error_code,
            "at": at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        clip_id = d.pop("clip_id")

        kind = WorkerRecentTaskKind(d.pop("kind"))




        outcome = WorkerRecentTaskOutcome(d.pop("outcome"))




        error_code = d.pop("error_code")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        worker_recent_task = cls(
            job_id=job_id,
            clip_id=clip_id,
            kind=kind,
            outcome=outcome,
            error_code=error_code,
            at=at,
        )


        worker_recent_task.additional_properties = d
        return worker_recent_task

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
