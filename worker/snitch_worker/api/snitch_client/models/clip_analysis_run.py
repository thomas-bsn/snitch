from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_analysis_run_forced_steps_item import ClipAnalysisRunForcedStepsItem
from ..models.clip_analysis_run_mode import ClipAnalysisRunMode
from ..models.clip_analysis_run_scope import ClipAnalysisRunScope
from ..models.clip_analysis_run_status import ClipAnalysisRunStatus
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.clip_analysis_step import ClipAnalysisStep





T = TypeVar("T", bound="ClipAnalysisRun")



@_attrs_define
class ClipAnalysisRun:
    """
        Attributes:
            id (int):
            clip_id (int):
            scope (ClipAnalysisRunScope):
            status (ClipAnalysisRunStatus):
            started_by (int | None):
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            finished_at (datetime.datetime | None):
            mode (ClipAnalysisRunMode):
            forced_steps (list[ClipAnalysisRunForcedStepsItem]):
            confirmed_replace_corrections (bool):
            confirmed_replace_validations (bool):
            confirmed_stake_at (datetime.datetime | None): Instant jusqu'auquel le travail humain en jeu a été montré et
                confirmé : la réanalyse ne rend obsolète que ce qui existait alors (nul sans étape relancée).
            pending_voices (int): Voix du clip encore inconnues ou ambiguës (liste de vérification), lues en direct : une
                analyse terminée qui en laisse est seulement partiellement identifiée.
            steps (list[ClipAnalysisStep]):
     """

    id: int
    clip_id: int
    scope: ClipAnalysisRunScope
    status: ClipAnalysisRunStatus
    started_by: int | None
    created_at: datetime.datetime
    updated_at: datetime.datetime
    finished_at: datetime.datetime | None
    mode: ClipAnalysisRunMode
    forced_steps: list[ClipAnalysisRunForcedStepsItem]
    confirmed_replace_corrections: bool
    confirmed_replace_validations: bool
    confirmed_stake_at: datetime.datetime | None
    pending_voices: int
    steps: list[ClipAnalysisStep]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.clip_analysis_step import ClipAnalysisStep # noqa: PLC0415
        id = self.id

        clip_id = self.clip_id

        scope = self.scope.value

        status = self.status.value

        started_by: int | None
        started_by = self.started_by

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        mode = self.mode.value

        forced_steps = []
        for forced_steps_item_data in self.forced_steps:
            forced_steps_item = forced_steps_item_data.value
            forced_steps.append(forced_steps_item)



        confirmed_replace_corrections = self.confirmed_replace_corrections

        confirmed_replace_validations = self.confirmed_replace_validations

        confirmed_stake_at: None | str
        if isinstance(self.confirmed_stake_at, datetime.datetime):
            confirmed_stake_at = self.confirmed_stake_at.isoformat()
        else:
            confirmed_stake_at = self.confirmed_stake_at

        pending_voices = self.pending_voices

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "clip_id": clip_id,
            "scope": scope,
            "status": status,
            "started_by": started_by,
            "created_at": created_at,
            "updated_at": updated_at,
            "finished_at": finished_at,
            "mode": mode,
            "forced_steps": forced_steps,
            "confirmed_replace_corrections": confirmed_replace_corrections,
            "confirmed_replace_validations": confirmed_replace_validations,
            "confirmed_stake_at": confirmed_stake_at,
            "pending_voices": pending_voices,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.clip_analysis_step import ClipAnalysisStep # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        clip_id = d.pop("clip_id")

        scope = ClipAnalysisRunScope(d.pop("scope"))




        status = ClipAnalysisRunStatus(d.pop("status"))




        def _parse_started_by(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        started_by = _parse_started_by(d.pop("started_by"))


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


        mode = ClipAnalysisRunMode(d.pop("mode"))




        forced_steps = []
        _forced_steps = d.pop("forced_steps")
        for forced_steps_item_data in (_forced_steps):
            forced_steps_item = ClipAnalysisRunForcedStepsItem(forced_steps_item_data)



            forced_steps.append(forced_steps_item)


        confirmed_replace_corrections = d.pop("confirmed_replace_corrections")

        confirmed_replace_validations = d.pop("confirmed_replace_validations")

        def _parse_confirmed_stake_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                confirmed_stake_at_type_0 = datetime.datetime.fromisoformat(data)



                return confirmed_stake_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        confirmed_stake_at = _parse_confirmed_stake_at(d.pop("confirmed_stake_at"))


        pending_voices = d.pop("pending_voices")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = ClipAnalysisStep.from_dict(steps_item_data)



            steps.append(steps_item)


        clip_analysis_run = cls(
            id=id,
            clip_id=clip_id,
            scope=scope,
            status=status,
            started_by=started_by,
            created_at=created_at,
            updated_at=updated_at,
            finished_at=finished_at,
            mode=mode,
            forced_steps=forced_steps,
            confirmed_replace_corrections=confirmed_replace_corrections,
            confirmed_replace_validations=confirmed_replace_validations,
            confirmed_stake_at=confirmed_stake_at,
            pending_voices=pending_voices,
            steps=steps,
        )


        clip_analysis_run.additional_properties = d
        return clip_analysis_run

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
