from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.audio_plan_state import AudioPlanState





T = TypeVar("T", bound="AudioSelectionStarted")



@_attrs_define
class AudioSelectionStarted:
    """
        Attributes:
            state (AudioPlanState):
            worker_compatible (bool):
     """

    state: AudioPlanState
    worker_compatible: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_plan_state import AudioPlanState # noqa: PLC0415
        state = self.state.to_dict()

        worker_compatible = self.worker_compatible


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "state": state,
            "worker_compatible": worker_compatible,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_plan_state import AudioPlanState # noqa: PLC0415
        d = dict(src_dict)
        state = AudioPlanState.from_dict(d.pop("state"))




        worker_compatible = d.pop("worker_compatible")

        audio_selection_started = cls(
            state=state,
            worker_compatible=worker_compatible,
        )

        return audio_selection_started
