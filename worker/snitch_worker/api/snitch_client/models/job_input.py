from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="JobInput")



@_attrs_define
class JobInput:
    """
        Attributes:
            clip_id (int):
            step (str):
            input_version (str):
            priority (int | Unset):  Default: 0.
            max_attempts (int | Unset):  Default: 3.
            requested_model_id (str | Unset): Modèle d'empreinte du catalogue demandé explicitement (benchmark), uniquement
                pour une étape voice.fingerprint.*. Sans lui, le modèle actif s'applique. Le travail n'est attribué qu'à un
                Worker qui a CE modèle prêt ; ses empreintes restent dans un groupe de benchmark, jamais comparé à la
                production.
     """

    clip_id: int
    step: str
    input_version: str
    priority: int | Unset = 0
    max_attempts: int | Unset = 3
    requested_model_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clip_id = self.clip_id

        step = self.step

        input_version = self.input_version

        priority = self.priority

        max_attempts = self.max_attempts

        requested_model_id = self.requested_model_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clip_id": clip_id,
            "step": step,
            "input_version": input_version,
        })
        if priority is not UNSET:
            field_dict["priority"] = priority
        if max_attempts is not UNSET:
            field_dict["max_attempts"] = max_attempts
        if requested_model_id is not UNSET:
            field_dict["requested_model_id"] = requested_model_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        step = d.pop("step")

        input_version = d.pop("input_version")

        priority = d.pop("priority", UNSET)

        max_attempts = d.pop("max_attempts", UNSET)

        requested_model_id = d.pop("requested_model_id", UNSET)

        job_input = cls(
            clip_id=clip_id,
            step=step,
            input_version=input_version,
            priority=priority,
            max_attempts=max_attempts,
            requested_model_id=requested_model_id,
        )

        return job_input
