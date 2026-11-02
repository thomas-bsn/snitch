from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TaskCardUnknownStepType0")



@_attrs_define
class TaskCardUnknownStepType0:
    """ Tâche non terminée d'une étape inconnue rattachée à la carte : alerte et erreur visibles.

        Attributes:
            step (str):
            state (str):
            last_error (str):
            job_id (int | Unset):
     """

    step: str
    state: str
    last_error: str
    job_id: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        step = self.step

        state = self.state

        last_error = self.last_error

        job_id = self.job_id


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "step": step,
            "state": state,
            "last_error": last_error,
        })
        if job_id is not UNSET:
            field_dict["job_id"] = job_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step = d.pop("step")

        state = d.pop("state")

        last_error = d.pop("last_error")

        job_id = d.pop("job_id", UNSET)

        task_card_unknown_step_type_0 = cls(
            step=step,
            state=state,
            last_error=last_error,
            job_id=job_id,
        )


        task_card_unknown_step_type_0.additional_properties = d
        return task_card_unknown_step_type_0

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
