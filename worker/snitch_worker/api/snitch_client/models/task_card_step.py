from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.task_card_step_key import TaskCardStepKey
from ..models.task_card_step_state import TaskCardStepState
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TaskCardStep")



@_attrs_define
class TaskCardStep:
    """
        Attributes:
            key (TaskCardStepKey):
            state (TaskCardStepState):
            job_id (int | None | Unset):
            job_status (None | str | Unset):
     """

    key: TaskCardStepKey
    state: TaskCardStepState
    job_id: int | None | Unset = UNSET
    job_status: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        state = self.state.value

        job_id: int | None | Unset
        if isinstance(self.job_id, Unset):
            job_id = UNSET
        else:
            job_id = self.job_id

        job_status: None | str | Unset
        if isinstance(self.job_status, Unset):
            job_status = UNSET
        else:
            job_status = self.job_status


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "key": key,
            "state": state,
        })
        if job_id is not UNSET:
            field_dict["job_id"] = job_id
        if job_status is not UNSET:
            field_dict["job_status"] = job_status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = TaskCardStepKey(d.pop("key"))




        state = TaskCardStepState(d.pop("state"))




        def _parse_job_id(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        job_id = _parse_job_id(d.pop("job_id", UNSET))


        def _parse_job_status(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        job_status = _parse_job_status(d.pop("job_status", UNSET))


        task_card_step = cls(
            key=key,
            state=state,
            job_id=job_id,
            job_status=job_status,
        )


        task_card_step.additional_properties = d
        return task_card_step

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
