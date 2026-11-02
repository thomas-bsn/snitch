from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.clip_analysis_step_status import ClipAnalysisStepStatus
from ..models.clip_analysis_step_step import ClipAnalysisStepStep
from typing import cast
import datetime






T = TypeVar("T", bound="ClipAnalysisStep")



@_attrs_define
class ClipAnalysisStep:
    """
        Attributes:
            step (ClipAnalysisStepStep):
            status (ClipAnalysisStepStatus):
            reason (str): Code stable : awaiting_preparation, awaiting_previous_step, clip_not_ready,
                audio_track_selection_required, clip_ready, awaiting_worker, running, retry_scheduled, completed,
                reused_existing, kept_for_corrections, job_dead, job_cancelled, job_failed, outside_scope,
                transcription_required, awaiting_transcription, awaiting_indexer, semantic_search_unavailable,
                semantic_index_failed, microphone_track, microphone_direct_attribution, no_speakers, awaiting_fingerprints,
                matching_failed, upstream_failed.
            job_id (int | None):
            attempt (int):
            started_at (datetime.datetime | None):
            finished_at (datetime.datetime | None):
            updated_at (datetime.datetime):
     """

    step: ClipAnalysisStepStep
    status: ClipAnalysisStepStatus
    reason: str
    job_id: int | None
    attempt: int
    started_at: datetime.datetime | None
    finished_at: datetime.datetime | None
    updated_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        step = self.step.value

        status = self.status.value

        reason = self.reason

        job_id: int | None
        job_id = self.job_id

        attempt = self.attempt

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


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "step": step,
            "status": status,
            "reason": reason,
            "job_id": job_id,
            "attempt": attempt,
            "started_at": started_at,
            "finished_at": finished_at,
            "updated_at": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
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




        clip_analysis_step = cls(
            step=step,
            status=status,
            reason=reason,
            job_id=job_id,
            attempt=attempt,
            started_at=started_at,
            finished_at=finished_at,
            updated_at=updated_at,
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
