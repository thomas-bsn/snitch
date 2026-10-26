from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.setup_step_input_state import SetupStepInputState
from ..models.setup_step_input_step import SetupStepInputStep






T = TypeVar("T", bound="SetupStepInput")



@_attrs_define
class SetupStepInput:
    """
        Attributes:
            step (SetupStepInputStep):
            state (SetupStepInputState):
     """

    step: SetupStepInputStep
    state: SetupStepInputState





    def to_dict(self) -> dict[str, Any]:
        step = self.step.value

        state = self.state.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "step": step,
            "state": state,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step = SetupStepInputStep(d.pop("step"))




        state = SetupStepInputState(d.pop("state"))




        setup_step_input = cls(
            step=step,
            state=state,
        )

        return setup_step_input
