from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.job_event_actor import JobEventActor
from ..models.job_event_event import JobEventEvent
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AnalysisStepEvent")



@_attrs_define
class AnalysisStepEvent:
    """
        Attributes:
            id (int):
            event (JobEventEvent):
            cycle (int):
            attempt (int | None):
            actor (JobEventActor):
            actor_id (str):
            error_code (str): Cause d'échec (voir FailureCode) ; vide pour un événement sans échec.
            created_at (datetime.datetime):
            step (str | Unset): Étape du job, présente avec un diagnostic.
            provider (str | Unset):
            model (str | Unset):
            revision (str | Unset):
            http_status (int | Unset):
            duration_ms (int | Unset):
            recommended_action (str | Unset): Code de l'action de reprise conseillée, décidée par le serveur ; absent sans
                cause connue.
            worker_name (str | Unset): Nom du Worker de l'événement, quand il est encore enregistré ; jamais un chemin.
     """

    id: int
    event: JobEventEvent
    cycle: int
    attempt: int | None
    actor: JobEventActor
    actor_id: str
    error_code: str
    created_at: datetime.datetime
    step: str | Unset = UNSET
    provider: str | Unset = UNSET
    model: str | Unset = UNSET
    revision: str | Unset = UNSET
    http_status: int | Unset = UNSET
    duration_ms: int | Unset = UNSET
    recommended_action: str | Unset = UNSET
    worker_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        event = self.event.value

        cycle = self.cycle

        attempt: int | None
        attempt = self.attempt

        actor = self.actor.value

        actor_id = self.actor_id

        error_code = self.error_code

        created_at = self.created_at.isoformat()

        step = self.step

        provider = self.provider

        model = self.model

        revision = self.revision

        http_status = self.http_status

        duration_ms = self.duration_ms

        recommended_action = self.recommended_action

        worker_name = self.worker_name


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "event": event,
            "cycle": cycle,
            "attempt": attempt,
            "actor": actor,
            "actor_id": actor_id,
            "error_code": error_code,
            "created_at": created_at,
        })
        if step is not UNSET:
            field_dict["step"] = step
        if provider is not UNSET:
            field_dict["provider"] = provider
        if model is not UNSET:
            field_dict["model"] = model
        if revision is not UNSET:
            field_dict["revision"] = revision
        if http_status is not UNSET:
            field_dict["http_status"] = http_status
        if duration_ms is not UNSET:
            field_dict["duration_ms"] = duration_ms
        if recommended_action is not UNSET:
            field_dict["recommended_action"] = recommended_action
        if worker_name is not UNSET:
            field_dict["worker_name"] = worker_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        event = JobEventEvent(d.pop("event"))




        cycle = d.pop("cycle")

        def _parse_attempt(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        attempt = _parse_attempt(d.pop("attempt"))


        actor = JobEventActor(d.pop("actor"))




        actor_id = d.pop("actor_id")

        error_code = d.pop("error_code")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        step = d.pop("step", UNSET)

        provider = d.pop("provider", UNSET)

        model = d.pop("model", UNSET)

        revision = d.pop("revision", UNSET)

        http_status = d.pop("http_status", UNSET)

        duration_ms = d.pop("duration_ms", UNSET)

        recommended_action = d.pop("recommended_action", UNSET)

        worker_name = d.pop("worker_name", UNSET)

        analysis_step_event = cls(
            id=id,
            event=event,
            cycle=cycle,
            attempt=attempt,
            actor=actor,
            actor_id=actor_id,
            error_code=error_code,
            created_at=created_at,
            step=step,
            provider=provider,
            model=model,
            revision=revision,
            http_status=http_status,
            duration_ms=duration_ms,
            recommended_action=recommended_action,
            worker_name=worker_name,
        )


        analysis_step_event.additional_properties = d
        return analysis_step_event

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
