from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.notification_kind import NotificationKind
from ..models.notification_state import NotificationState
from typing import cast
import datetime






T = TypeVar("T", bound="Notification")



@_attrs_define
class Notification:
    """
        Attributes:
            id (int):
            kind (NotificationKind): Voix à vérifier : une notification active par clip, résolue automatiquement quand
                toutes ses voix sont traitées. Incidents techniques (administrateurs) : une notification active par cause,
                résolue automatiquement au retour du worker ou du service ; un non-administrateur ne reçoit que
                technical_incident, message générique sans détail.
            state (NotificationState): Partagé : une action résolue l'est pour tous.
            clip_id (int | None):
            clip_name (str):
            pending_count (int): Voix restant à décider, ou jobs concernés par l'incident.
            read (bool): État de lecture du demandeur, indépendant de la résolution.
            created_at (datetime.datetime):
            updated_at (datetime.datetime):
            resolved_at (datetime.datetime | None):
            job_id (int | None): Incident : dernier job concerné (administrateurs uniquement).
            step (str): Incident : étape concernée (administrateurs uniquement), jamais de chemin ni de secret.
            error_code (str): Incident : code d'erreur normalisé (provider_unavailable, model_missing, invalid_input,
                worker_failed, retries_exhausted, transcription_version_mismatch, transcription_not_validated), jamais l'erreur
                brute.
            generic (bool): Vrai pour le message générique sûr vu par un non-administrateur.
     """

    id: int
    kind: NotificationKind
    state: NotificationState
    clip_id: int | None
    clip_name: str
    pending_count: int
    read: bool
    created_at: datetime.datetime
    updated_at: datetime.datetime
    resolved_at: datetime.datetime | None
    job_id: int | None
    step: str
    error_code: str
    generic: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value

        state = self.state.value

        clip_id: int | None
        clip_id = self.clip_id

        clip_name = self.clip_name

        pending_count = self.pending_count

        read = self.read

        created_at = self.created_at.isoformat()

        updated_at = self.updated_at.isoformat()

        resolved_at: None | str
        if isinstance(self.resolved_at, datetime.datetime):
            resolved_at = self.resolved_at.isoformat()
        else:
            resolved_at = self.resolved_at

        job_id: int | None
        job_id = self.job_id

        step = self.step

        error_code = self.error_code

        generic = self.generic


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "kind": kind,
            "state": state,
            "clip_id": clip_id,
            "clip_name": clip_name,
            "pending_count": pending_count,
            "read": read,
            "created_at": created_at,
            "updated_at": updated_at,
            "resolved_at": resolved_at,
            "job_id": job_id,
            "step": step,
            "error_code": error_code,
            "generic": generic,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = NotificationKind(d.pop("kind"))




        state = NotificationState(d.pop("state"))




        def _parse_clip_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        clip_id = _parse_clip_id(d.pop("clip_id"))


        clip_name = d.pop("clip_name")

        pending_count = d.pop("pending_count")

        read = d.pop("read")

        created_at = datetime.datetime.fromisoformat(d.pop("created_at"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updated_at"))




        def _parse_resolved_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                resolved_at_type_0 = datetime.datetime.fromisoformat(data)



                return resolved_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        resolved_at = _parse_resolved_at(d.pop("resolved_at"))


        def _parse_job_id(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        job_id = _parse_job_id(d.pop("job_id"))


        step = d.pop("step")

        error_code = d.pop("error_code")

        generic = d.pop("generic")

        notification = cls(
            id=id,
            kind=kind,
            state=state,
            clip_id=clip_id,
            clip_name=clip_name,
            pending_count=pending_count,
            read=read,
            created_at=created_at,
            updated_at=updated_at,
            resolved_at=resolved_at,
            job_id=job_id,
            step=step,
            error_code=error_code,
            generic=generic,
        )


        notification.additional_properties = d
        return notification

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
