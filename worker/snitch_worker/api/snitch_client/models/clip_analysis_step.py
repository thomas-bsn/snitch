from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_analysis_step_status import ClipAnalysisStepStatus
from ..models.clip_analysis_step_step import ClipAnalysisStepStep
from ..models.failure_code import FailureCode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.analysis_step_event import AnalysisStepEvent





T = TypeVar("T", bound="ClipAnalysisStep")



@_attrs_define
class ClipAnalysisStep:
    """
        Attributes:
            step (ClipAnalysisStepStep):
            status (ClipAnalysisStepStatus):
            reason (str): Code stable : awaiting_preparation, awaiting_previous_step, clip_not_ready,
                audio_track_selection_required, clip_ready, awaiting_worker, awaiting_compatible_worker, queue_paused,
                transcription_version_mismatch, transcription_not_validated, run_cancelled, running, retry_scheduled, completed,
                reused_existing, kept_for_corrections, job_dead, job_cancelled, job_failed, outside_scope,
                transcription_required, awaiting_transcription, awaiting_indexer, semantic_search_unavailable,
                semantic_index_failed, microphone_track, microphone_direct_attribution, no_speakers, awaiting_fingerprints,
                matching_failed, upstream_failed.
            job_id (int | None):
            attempt (int):
            input_version (None | str): Version d'entrée du job suivi par l'étape (historique).
            profile (None | str): Profil (pipeline) du job suivi par l'étape.
            forced (bool): Étape relancée par une réanalyse, malgré un résultat valide.
            started_at (datetime.datetime | None):
            finished_at (datetime.datetime | None):
            updated_at (datetime.datetime):
            failure_code (FailureCode | Unset): Catalogue fermé des Causes d'échec d'un job. Un code hors catalogue envoyé
                par un ancien Worker est enregistré comme worker_failed.
            recommended_action (str | Unset): File de tâches seulement : code de l'action conseillée, décidé par le serveur.
                Étape en échec : action de la Cause d'échec (catalogue des jobs). Attente de Worker (awaiting_worker,
                awaiting_compatible_worker) d'un job prêt : start_worker ; jamais pour une reprise différée.
            events (list[AnalysisStepEvent] | Unset): Historique des analyses seulement : événements persistés du job suivi
                par l'étape (tentatives, reprises, échecs), du plus ancien au plus récent ; absent ailleurs ou sans job.
     """

    step: ClipAnalysisStepStep
    status: ClipAnalysisStepStatus
    reason: str
    job_id: int | None
    attempt: int
    input_version: None | str
    profile: None | str
    forced: bool
    started_at: datetime.datetime | None
    finished_at: datetime.datetime | None
    updated_at: datetime.datetime
    failure_code: FailureCode | Unset = UNSET
    recommended_action: str | Unset = UNSET
    events: list[AnalysisStepEvent] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.analysis_step_event import AnalysisStepEvent # noqa: PLC0415
        step = self.step.value

        status = self.status.value

        reason = self.reason

        job_id: int | None
        job_id = self.job_id

        attempt = self.attempt

        input_version: None | str
        input_version = self.input_version

        profile: None | str
        profile = self.profile

        forced = self.forced

        started_at: None | str
        if isinstance(self.started_at, datetime.datetime):
            started_at = self.started_at.isoformat()
        else:
            started_at = self.started_at

        finished_at: None | str
        if isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        updated_at = self.updated_at.isoformat()

        failure_code: str | Unset = UNSET
        if not isinstance(self.failure_code, Unset):
            failure_code = self.failure_code.value


        recommended_action = self.recommended_action

        events: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.events, Unset):
            events = []
            for events_item_data in self.events:
                events_item = events_item_data.to_dict()
                events.append(events_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "step": step,
            "status": status,
            "reason": reason,
            "job_id": job_id,
            "attempt": attempt,
            "input_version": input_version,
            "profile": profile,
            "forced": forced,
            "started_at": started_at,
            "finished_at": finished_at,
            "updated_at": updated_at,
        })
        if failure_code is not UNSET:
            field_dict["failure_code"] = failure_code
        if recommended_action is not UNSET:
            field_dict["recommended_action"] = recommended_action
        if events is not UNSET:
            field_dict["events"] = events

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.analysis_step_event import AnalysisStepEvent # noqa: PLC0415
        d = dict(src_dict)
        step = ClipAnalysisStepStep(d.pop("step"))




        status = ClipAnalysisStepStatus(d.pop("status"))




        reason = d.pop("reason")

        def _parse_job_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        job_id = _parse_job_id(d.pop("job_id"))


        attempt = d.pop("attempt")

        def _parse_input_version(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        input_version = _parse_input_version(d.pop("input_version"))


        def _parse_profile(data: object) -> None | str:
            if data is None:
                return data
            return cast(None | str, data)

        profile = _parse_profile(d.pop("profile"))


        forced = d.pop("forced")

        def _parse_started_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                started_at_type_0 = datetime.datetime.fromisoformat(data)



                return started_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        started_at = _parse_started_at(d.pop("started_at"))


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


        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        _failure_code = d.pop("failure_code", UNSET)
        failure_code: FailureCode | Unset
        if isinstance(_failure_code,  Unset):
            failure_code = UNSET
        else:
            failure_code = FailureCode(_failure_code)




        recommended_action = d.pop("recommended_action", UNSET)

        _events = d.pop("events", UNSET)
        events: list[AnalysisStepEvent] | Unset = UNSET
        if _events is not UNSET:
            events = []
            for events_item_data in _events:
                events_item = AnalysisStepEvent.from_dict(events_item_data)



                events.append(events_item)


        clip_analysis_step = cls(
            step=step,
            status=status,
            reason=reason,
            job_id=job_id,
            attempt=attempt,
            input_version=input_version,
            profile=profile,
            forced=forced,
            started_at=started_at,
            finished_at=finished_at,
            updated_at=updated_at,
            failure_code=failure_code,
            recommended_action=recommended_action,
            events=events,
        )


        clip_analysis_step.additional_properties = d
        return clip_analysis_step

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
