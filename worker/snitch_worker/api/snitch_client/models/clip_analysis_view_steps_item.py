from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_analysis_view_steps_item_status import ClipAnalysisViewStepsItemStatus
from ..models.clip_analysis_view_steps_item_step import ClipAnalysisViewStepsItemStep
from typing import cast
import datetime






T = TypeVar("T", bound="ClipAnalysisViewStepsItem")



@_attrs_define
class ClipAnalysisViewStepsItem:
    """
        Attributes:
            step (ClipAnalysisViewStepsItemStep):
            status (ClipAnalysisViewStepsItemStatus):
            reason (str):
            updated_at (datetime.datetime):
     """

    step: ClipAnalysisViewStepsItemStep
    status: ClipAnalysisViewStepsItemStatus
    reason: str
    updated_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        step = self.step.value

        status = self.status.value

        reason = self.reason

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "step": step,
            "status": status,
            "reason": reason,
            "updated_at": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step = ClipAnalysisViewStepsItemStep(d.pop("step"))




        status = ClipAnalysisViewStepsItemStatus(d.pop("status"))




        reason = d.pop("reason")

        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        clip_analysis_view_steps_item = cls(
            step=step,
            status=status,
            reason=reason,
            updated_at=updated_at,
        )


        clip_analysis_view_steps_item.additional_properties = d
        return clip_analysis_view_steps_item

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
