from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_step_status import VoiceStepStatus






T = TypeVar("T", bound="VoiceStep")



@_attrs_define
class VoiceStep:
    """
        Attributes:
            job_id (int):
            step (str):
            status (VoiceStepStatus):
            input_version (str):
            attempts (int):
            max_attempts (int):
            last_error (str): Code d'erreur expurgé, jamais de détail du fournisseur.
     """

    job_id: int
    step: str
    status: VoiceStepStatus
    input_version: str
    attempts: int
    max_attempts: int
    last_error: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        job_id = self.job_id

        step = self.step

        status = self.status.value

        input_version = self.input_version

        attempts = self.attempts

        max_attempts = self.max_attempts

        last_error = self.last_error


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "job_id": job_id,
            "step": step,
            "status": status,
            "input_version": input_version,
            "attempts": attempts,
            "max_attempts": max_attempts,
            "last_error": last_error,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        job_id = d.pop("job_id")

        step = d.pop("step")

        status = VoiceStepStatus(d.pop("status"))




        input_version = d.pop("input_version")

        attempts = d.pop("attempts")

        max_attempts = d.pop("max_attempts")

        last_error = d.pop("last_error")

        voice_step = cls(
            job_id=job_id,
            step=step,
            status=status,
            input_version=input_version,
            attempts=attempts,
            max_attempts=max_attempts,
            last_error=last_error,
        )


        voice_step.additional_properties = d
        return voice_step

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
