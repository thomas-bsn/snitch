from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_install_state import WorkerInstallState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.worker_install_event import WorkerInstallEvent
  from ..models.worker_install_voice import WorkerInstallVoice





T = TypeVar("T", bound="WorkerInstall")



@_attrs_define
class WorkerInstall:
    """ Progression d'installation, indépendante de la transcription (transcription_config) et de la voix (voice).

        Attributes:
            state (WorkerInstallState): connected = premier pouls ; voice_ready = connecté et inférence synthétique réussie
                ; never_connected = aucun pouls après 24 h.
            failed_phase (str):
            cause_code (str):
            voice (WorkerInstallVoice):
            last_event (None | WorkerInstallEvent):
            last_activity_at (datetime.datetime | None):
            no_news (bool): Absence de nouvelles, distincte du dernier événement reçu.
            events (list[WorkerInstallEvent] | Unset): Historique récent (fiche détaillée seulement).
     """

    state: WorkerInstallState
    failed_phase: str
    cause_code: str
    voice: WorkerInstallVoice
    last_event: None | WorkerInstallEvent
    last_activity_at: datetime.datetime | None
    no_news: bool
    events: list[WorkerInstallEvent] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_install_event import WorkerInstallEvent # noqa: PLC0415
        from ..models.worker_install_voice import WorkerInstallVoice # noqa: PLC0415
        state = self.state.value

        failed_phase = self.failed_phase

        cause_code = self.cause_code

        voice = self.voice.to_dict()

        last_event: dict[str, Any] | None
        if isinstance(self.last_event, WorkerInstallEvent):
            last_event = self.last_event.to_dict()
        else:
            last_event = self.last_event

        last_activity_at: None | str
        if isinstance(self.last_activity_at, datetime.datetime):
            last_activity_at = self.last_activity_at.isoformat()
        else:
            last_activity_at = self.last_activity_at

        no_news = self.no_news

        events: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.events, Unset):
            events = []
            for events_item_data in self.events:
                events_item = events_item_data.to_dict()
                events.append(events_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "state": state,
            "failed_phase": failed_phase,
            "cause_code": cause_code,
            "voice": voice,
            "last_event": last_event,
            "last_activity_at": last_activity_at,
            "no_news": no_news,
        })
        if events is not UNSET:
            field_dict["events"] = events

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_install_event import WorkerInstallEvent # noqa: PLC0415
        from ..models.worker_install_voice import WorkerInstallVoice # noqa: PLC0415
        d = dict(src_dict)
        state = WorkerInstallState(d.pop("state"))




        failed_phase = d.pop("failed_phase")

        cause_code = d.pop("cause_code")

        voice = WorkerInstallVoice.from_dict(d.pop("voice"))




        def _parse_last_event(data: object) -> None | WorkerInstallEvent:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                last_event_type_1 = WorkerInstallEvent.from_dict(data)



                return last_event_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | WorkerInstallEvent, data)

        last_event = _parse_last_event(d.pop("last_event"))


        def _parse_last_activity_at(data: object) -> datetime.datetime | None:
            if data is None:
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_activity_at_type_0 = datetime.datetime.fromisoformat(data)



                return last_activity_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None, data)

        last_activity_at = _parse_last_activity_at(d.pop("last_activity_at"))


        no_news = d.pop("no_news")

        _events = d.pop("events", UNSET)
        events: list[WorkerInstallEvent] | Unset = UNSET
        if _events is not UNSET:
            events = []
            for events_item_data in _events:
                events_item = WorkerInstallEvent.from_dict(events_item_data)



                events.append(events_item)


        worker_install = cls(
            state=state,
            failed_phase=failed_phase,
            cause_code=cause_code,
            voice=voice,
            last_event=last_event,
            last_activity_at=last_activity_at,
            no_news=no_news,
            events=events,
        )


        worker_install.additional_properties = d
        return worker_install

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
