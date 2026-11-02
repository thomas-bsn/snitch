from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_analysis_view_scope import ClipAnalysisViewScope
from ..models.clip_analysis_view_status import ClipAnalysisViewStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.clip_analysis_view_steps_item import ClipAnalysisViewStepsItem





T = TypeVar("T", bound="ClipAnalysisView")



@_attrs_define
class ClipAnalysisView:
    """ Vue en lecture seule pour tout utilisateur qui peut ouvrir le clip : état et code de raison stable de chaque étape,
    sans job, Worker ni détail d'erreur.

        Attributes:
            clip_id (int):
            scope (ClipAnalysisViewScope):
            status (ClipAnalysisViewStatus):
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            finished_at (datetime.datetime | None):
            pending_voices (int):
            steps (list[ClipAnalysisViewStepsItem]):
     """

    clip_id: int
    scope: ClipAnalysisViewScope
    status: ClipAnalysisViewStatus
    created_at: datetime.datetime
    updated_at: datetime.datetime
    finished_at: datetime.datetime | None
    pending_voices: int
    steps: list[ClipAnalysisViewStepsItem]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_view_steps_item import ClipAnalysisViewStepsItem # noqa: PLC0415
        clip_id = self.clip_id

        scope = self.scope.value

        status = self.status.value

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        pending_voices = self.pending_voices

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "clip_id": clip_id,
            "scope": scope,
            "status": status,
            "created_at": created_at,
            "updated_at": updated_at,
            "finished_at": finished_at,
            "pending_voices": pending_voices,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_view_steps_item import ClipAnalysisViewStepsItem # noqa: PLC0415
        d = dict(src_dict)
        clip_id = d.pop("clip_id")

        scope = ClipAnalysisViewScope(d.pop("scope"))




        status = ClipAnalysisViewStatus(d.pop("status"))




        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        def _parse_finished_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                finished_at_type_0 = datetime.datetime.fromisoformat(data)



                return finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        finished_at = _parse_finished_at(d.pop("finished_at"))


        pending_voices = d.pop("pending_voices")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = ClipAnalysisViewStepsItem.from_dict(steps_item_data)



            steps.append(steps_item)


        clip_analysis_view = cls(
            clip_id=clip_id,
            scope=scope,
            status=status,
            created_at=created_at,
            updated_at=updated_at,
            finished_at=finished_at,
            pending_voices=pending_voices,
            steps=steps,
        )


        clip_analysis_view.additional_properties = d
        return clip_analysis_view

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
