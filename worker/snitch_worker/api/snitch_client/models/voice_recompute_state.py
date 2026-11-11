from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.voice_recompute_run import VoiceRecomputeRun





T = TypeVar("T", bound="VoiceRecomputeState")



@_attrs_define
class VoiceRecomputeState:
    """
        Attributes:
            transition_open (bool):
            run (VoiceRecomputeRun | Unset):
     """

    transition_open: bool
    run: VoiceRecomputeRun | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.voice_recompute_run import VoiceRecomputeRun # noqa: PLC0415
        transition_open = self.transition_open

        run: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run, Unset):
            run = self.run.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "transition_open": transition_open,
        })
        if run is not UNSET:
            field_dict["run"] = run

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.voice_recompute_run import VoiceRecomputeRun # noqa: PLC0415
        d = dict(src_dict)
        transition_open = d.pop("transition_open")

        _run = d.pop("run", UNSET)
        run: VoiceRecomputeRun | Unset
        if isinstance(_run,  Unset):
            run = UNSET
        else:
            run = VoiceRecomputeRun.from_dict(_run)




        voice_recompute_state = cls(
            transition_open=transition_open,
            run=run,
        )

        return voice_recompute_state
