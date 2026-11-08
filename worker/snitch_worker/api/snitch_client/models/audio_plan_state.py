from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audio_loss_preview import AudioLossPreview
  from ..models.audio_plan import AudioPlan
  from ..models.audio_plan_child import AudioPlanChild
  from ..models.audio_track_diagnostic import AudioTrackDiagnostic





T = TypeVar("T", bound="AudioPlanState")



@_attrs_define
class AudioPlanState:
    """
        Attributes:
            plan (AudioPlan):
            diagnostics (list[AudioTrackDiagnostic]):
            children (list[AudioPlanChild]):
            loss_preview (AudioLossPreview | Unset):
     """

    plan: AudioPlan
    diagnostics: list[AudioTrackDiagnostic]
    children: list[AudioPlanChild]
    loss_preview: AudioLossPreview | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        from ..models.audio_plan import AudioPlan # noqa: PLC0415
        from ..models.audio_plan_child import AudioPlanChild # noqa: PLC0415
        from ..models.audio_track_diagnostic import AudioTrackDiagnostic # noqa: PLC0415
        plan = self.plan.to_dict()

        diagnostics = []
        for diagnostics_item_data in self.diagnostics:
            diagnostics_item = diagnostics_item_data.to_dict()
            diagnostics.append(diagnostics_item)



        children = []
        for children_item_data in self.children:
            children_item = children_item_data.to_dict()
            children.append(children_item)



        loss_preview: dict[str, Any] | Unset = UNSET
        if not isinstance(self.loss_preview, Unset):
            loss_preview = self.loss_preview.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "plan": plan,
            "diagnostics": diagnostics,
            "children": children,
        })
        if loss_preview is not UNSET:
            field_dict["loss_preview"] = loss_preview

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audio_loss_preview import AudioLossPreview # noqa: PLC0415
        from ..models.audio_plan import AudioPlan # noqa: PLC0415
        from ..models.audio_plan_child import AudioPlanChild # noqa: PLC0415
        from ..models.audio_track_diagnostic import AudioTrackDiagnostic # noqa: PLC0415
        d = dict(src_dict)
        plan = AudioPlan.from_dict(d.pop("plan"))




        diagnostics = []
        _diagnostics = d.pop("diagnostics")
        for diagnostics_item_data in (_diagnostics):
            diagnostics_item = AudioTrackDiagnostic.from_dict(diagnostics_item_data)



            diagnostics.append(diagnostics_item)


        children = []
        _children = d.pop("children")
        for children_item_data in (_children):
            children_item = AudioPlanChild.from_dict(children_item_data)



            children.append(children_item)


        _loss_preview = d.pop("loss_preview", UNSET)
        loss_preview: AudioLossPreview | Unset
        if isinstance(_loss_preview,  Unset):
            loss_preview = UNSET
        else:
            loss_preview = AudioLossPreview.from_dict(_loss_preview)




        audio_plan_state = cls(
            plan=plan,
            diagnostics=diagnostics,
            children=children,
            loss_preview=loss_preview,
        )

        return audio_plan_state
