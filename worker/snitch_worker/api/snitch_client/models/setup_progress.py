from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.setup_step_state import SetupStepState





T = TypeVar("T", bound="SetupProgress")



@_attrs_define
class SetupProgress:
    """
        Attributes:
            steps (list[SetupStepState]):
            dismissed (bool): Le bandeau « Finir la configuration » a été fermé définitivement.
     """

    steps: list[SetupStepState]
    dismissed: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.setup_step_state import SetupStepState # noqa: PLC0415
        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        dismissed = self.dismissed


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "steps": steps,
            "dismissed": dismissed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.setup_step_state import SetupStepState # noqa: PLC0415
        d = dict(src_dict)
        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = SetupStepState.from_dict(steps_item_data)



            steps.append(steps_item)


        dismissed = d.pop("dismissed")

        setup_progress = cls(
            steps=steps,
            dismissed=dismissed,
        )


        setup_progress.additional_properties = d
        return setup_progress

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
