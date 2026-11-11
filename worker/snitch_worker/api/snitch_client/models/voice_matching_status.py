from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="VoiceMatchingStatus")



@_attrs_define
class VoiceMatchingStatus:
    """
        Attributes:
            pending (bool): Des demandes de rapprochement attendent (validation, correction, nouvelles empreintes, réglage).
            running (bool): Un passage de rapprochement est en cours.
            running_since (datetime.datetime | None): Début du passage en cours ; null sans passage en cours.
            last_finished_at (datetime.datetime | None): Fin du dernier passage réussi ; null s'il n'y en a jamais eu.
            last_duration_ms (int | None): Durée du dernier passage réussi, en millisecondes.
            last_requests (int | None): Nombre de demandes prises en compte par le dernier passage réussi : plusieurs
                décisions rapprochées font un seul passage.
            last_failed_at (datetime.datetime | None): Échec du dernier passage tenté, repris automatiquement ; null quand
                le dernier passage a réussi.
     """

    pending: bool
    running: bool
    running_since: datetime.datetime | None
    last_finished_at: datetime.datetime | None
    last_duration_ms: int | None
    last_requests: int | None
    last_failed_at: datetime.datetime | None
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        pending = self.pending

        running = self.running

        running_since: None | str
        if isinstance(self.running_since, datetime.datetime):
            running_since = self.running_since.isoformat()
        else:
            running_since = self.running_since

        last_finished_at: None | str
        if isinstance(self.last_finished_at, datetime.datetime):
            last_finished_at = self.last_finished_at.isoformat()
        else:
            last_finished_at = self.last_finished_at

        last_duration_ms: int | None
        last_duration_ms = self.last_duration_ms

        last_requests: int | None
        last_requests = self.last_requests

        last_failed_at: None | str
        if isinstance(self.last_failed_at, datetime.datetime):
            last_failed_at = self.last_failed_at.isoformat()
        else:
            last_failed_at = self.last_failed_at


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "pending": pending,
            "running": running,
            "running_since": running_since,
            "last_finished_at": last_finished_at,
            "last_duration_ms": last_duration_ms,
            "last_requests": last_requests,
            "last_failed_at": last_failed_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pending = d.pop("pending")

        running = d.pop("running")

        def _parse_running_since(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                running_since_type_0 = datetime.datetime.fromisoformat(data)



                return running_since_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        running_since = _parse_running_since(d.pop("running_since"))


        def _parse_last_finished_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_finished_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_finished_at = _parse_last_finished_at(d.pop("last_finished_at"))


        def _parse_last_duration_ms(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        last_duration_ms = _parse_last_duration_ms(d.pop("last_duration_ms"))


        def _parse_last_requests(data: object) -> int | None:
            if data is None:
                return data
            return cast(int | None, data)

        last_requests = _parse_last_requests(d.pop("last_requests"))


        def _parse_last_failed_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_failed_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_failed_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_failed_at = _parse_last_failed_at(d.pop("last_failed_at"))


        voice_matching_status = cls(
            pending=pending,
            running=running,
            running_since=running_since,
            last_finished_at=last_finished_at,
            last_duration_ms=last_duration_ms,
            last_requests=last_requests,
            last_failed_at=last_failed_at,
        )


        voice_matching_status.additional_properties = d
        return voice_matching_status

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
