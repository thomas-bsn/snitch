from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.audio_plan_input import AudioPlanInput





T = TypeVar("T", bound="AudioPlan")



@_attrs_define
class AudioPlan:
    """
        Attributes:
            id (int):
            run_id (int):
            clip_id (int):
            input_version (str):
            created_at (datetime.datetime):
            input_ (AudioPlanInput):
     """

    id: int
    run_id: int
    clip_id: int
    input_version: str
    created_at: datetime.datetime
    input_: AudioPlanInput





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        id = self.id

        run_id = self.run_id

        clip_id = self.clip_id

        input_version = self.input_version

        created_at = self.created_at.isoformat()

        input_ = self.input_.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "run_id": run_id,
            "clip_id": clip_id,
            "input_version": input_version,
            "created_at": created_at,
            "input": input_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_input import AudioPlanInput # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        run_id = d.pop("run_id")

        clip_id = d.pop("clip_id")

        input_version = d.pop("input_version")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        input_ = AudioPlanInput.from_dict(d.pop("input"))




        audio_plan = cls(
            id=id,
            run_id=run_id,
            clip_id=clip_id,
            input_version=input_version,
            created_at=created_at,
            input_=input_,
        )

        return audio_plan
