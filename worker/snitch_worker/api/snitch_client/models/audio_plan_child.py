from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audio_plan_child_stage import AudioPlanChildStage
from ..models.audio_plan_child_status import AudioPlanChildStatus






T = TypeVar("T", bound="AudioPlanChild")



@_attrs_define
class AudioPlanChild:
    """
        Attributes:
            plan_id (int):
            track_index (int):
            stage (AudioPlanChildStage):
            job_id (int):
            input_version (str):
            status (AudioPlanChildStatus):
            attempt (int):
     """

    plan_id: int
    track_index: int
    stage: AudioPlanChildStage
    job_id: int
    input_version: str
    status: AudioPlanChildStatus
    attempt: int





    def to_dict(self) -> dict[str, Any]:
        plan_id = self.plan_id

        track_index = self.track_index

        stage = self.stage.value

        job_id = self.job_id

        input_version = self.input_version

        status = self.status.value

        attempt = self.attempt


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "plan_id": plan_id,
            "track_index": track_index,
            "stage": stage,
            "job_id": job_id,
            "input_version": input_version,
            "status": status,
            "attempt": attempt,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        plan_id = d.pop("plan_id")

        track_index = d.pop("track_index")

        stage = AudioPlanChildStage(d.pop("stage"))




        job_id = d.pop("job_id")

        input_version = d.pop("input_version")

        status = AudioPlanChildStatus(d.pop("status"))




        attempt = d.pop("attempt")

        audio_plan_child = cls(
            plan_id=plan_id,
            track_index=track_index,
            stage=stage,
            job_id=job_id,
            input_version=input_version,
            status=status,
            attempt=attempt,
        )

        return audio_plan_child
