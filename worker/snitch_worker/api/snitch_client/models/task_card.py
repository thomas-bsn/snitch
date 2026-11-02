from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.task_card_column import TaskCardColumn
from ..models.task_card_current_step import TaskCardCurrentStep
from ..models.task_card_state import TaskCardState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.task_card_step import TaskCardStep





T = TypeVar("T", bound="TaskCard")



@_attrs_define
class TaskCard:
    """
        Attributes:
            clip_id (int):
            original_name (str):
            column (TaskCardColumn): search est réservée à la colonne Recherche, qui n'a pas encore d'état calculé.
            state (TaskCardState):
            current_step (TaskCardCurrentStep):
            steps (list[TaskCardStep]):
            updated_at (datetime.datetime):
            attempt (int):
            max_attempts (int):
            last_error (str):
            completed_at (datetime.datetime | None | Unset):
            worker (None | str | Unset):
            job_id (int | None | Unset):
            job_status (None | str | Unset): Statut brut de la tâche qui porte l'état courant.
            next_attempt_at (datetime.datetime | None | Unset):
     """

    clip_id: int
    original_name: str
    column: TaskCardColumn
    state: TaskCardState
    current_step: TaskCardCurrentStep
    steps: list[TaskCardStep]
    updated_at: datetime.datetime
    attempt: int
    max_attempts: int
    last_error: str
    completed_at: datetime.datetime | None | Unset = UNSET
    worker: None | str | Unset = UNSET
    job_id: int | None | Unset = UNSET
    job_status: None | str | Unset = UNSET
    next_attempt_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.task_card_step import TaskCardStep # noqa: PLC0415
        clip_id = self.clip_id

        original_name = self.original_name

        column = self.column.value

        state = self.state.value

        current_step = self.current_step.value

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        updated_at = self.updated_at.isoformat()

        attempt = self.attempt

        max_attempts = self.max_attempts

        last_error = self.last_error

        completed_at: None | str | Unset
        if isinstance(self.completed_at, Unset):
            completed_at = UNSET
        elif isinstance(self.completed_at, datetime.datetime):
            completed_at = self.completed_at.isoformat()
        else:
            completed_at = self.completed_at

        worker: None | str | Unset
        if isinstance(self.worker, Unset):
            worker = UNSET
        else:
            worker = self.worker

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

        next_attempt_at: None | str | Unset
        if isinstance(self.next_attempt_at, Unset):
            next_attempt_at = UNSET
        elif isinstance(self.next_attempt_at, datetime.datetime):
            next_attempt_at = self.next_attempt_at.isoformat()
        else:
            next_attempt_at = self.next_attempt_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "original_name": original_name,
            "column": column,
            "state": state,
            "current_step": current_step,
            "steps": steps,
            "updated_at": updated_at,
            "attempt": attempt,
            "max_attempts": max_attempts,
            "last_error": last_error,
        })
        if completed_at is not UNSET:
            field_dict["completed_at"] = completed_at
        if worker is not UNSET:
            field_dict["worker"] = worker
        if job_id is not UNSET:
            field_dict["job_id"] = job_id
        if job_status is not UNSET:
            field_dict["job_status"] = job_status
        if next_attempt_at is not UNSET:
            field_dict["next_attempt_at"] = next_attempt_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.task_card_step import TaskCardStep # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        original_name = d.pop("original_name")

        column = TaskCardColumn(d.pop("column"))




        state = TaskCardState(d.pop("state"))




        current_step = TaskCardCurrentStep(d.pop("current_step"))




        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = TaskCardStep.from_dict(steps_item_data)



            steps.append(steps_item)


        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        attempt = d.pop("attempt")

        max_attempts = d.pop("max_attempts")

        last_error = d.pop("last_error")

        def _parse_completed_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                completed_at_type_0 = datetime.datetime.fromisoformat(data)



                return completed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        completed_at = _parse_completed_at(d.pop("completed_at", UNSET))


        def _parse_worker(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        worker = _parse_worker(d.pop("worker", UNSET))


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


        def _parse_next_attempt_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                next_attempt_at_type_0 = datetime.datetime.fromisoformat(data)



                return next_attempt_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        next_attempt_at = _parse_next_attempt_at(d.pop("next_attempt_at", UNSET))


        task_card = cls(
            clip_id=clip_id,
            original_name=original_name,
            column=column,
            state=state,
            current_step=current_step,
            steps=steps,
            updated_at=updated_at,
            attempt=attempt,
            max_attempts=max_attempts,
            last_error=last_error,
            completed_at=completed_at,
            worker=worker,
            job_id=job_id,
            job_status=job_status,
            next_attempt_at=next_attempt_at,
        )


        task_card.additional_properties = d
        return task_card

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
