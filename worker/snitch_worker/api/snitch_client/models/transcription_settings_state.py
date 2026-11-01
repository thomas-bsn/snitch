from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transcription_settings_state_mode import TranscriptionSettingsStateMode
from ..models.transcription_settings_state_status import TranscriptionSettingsStateStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.worker_transcription_state import WorkerTranscriptionState





T = TypeVar("T", bound="TranscriptionSettingsState")



@_attrs_define
class TranscriptionSettingsState:
    """
        Attributes:
            mode (TranscriptionSettingsStateMode):
            status (TranscriptionSettingsStateStatus):
            endpoint (str):
            model (str):
            language (str):
            version (str):
            workers (list[WorkerTranscriptionState] | Unset):
     """

    mode: TranscriptionSettingsStateMode
    status: TranscriptionSettingsStateStatus
    endpoint: str
    model: str
    language: str
    version: str
    workers: list[WorkerTranscriptionState] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_transcription_state import WorkerTranscriptionState # noqa: PLC0415
        mode = self.mode.value

        status = self.status.value

        endpoint = self.endpoint

        model = self.model

        language = self.language

        version = self.version

        workers: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.workers, Unset):
            workers = []
            for workers_item_data in self.workers:
                workers_item = workers_item_data.to_dict()
                workers.append(workers_item)




        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "mode": mode,
            "status": status,
            "endpoint": endpoint,
            "model": model,
            "language": language,
            "version": version,
        })
        if workers is not UNSET:
            field_dict["workers"] = workers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_transcription_state import WorkerTranscriptionState # noqa: PLC0415
        d = dict(src_dict)
        mode = TranscriptionSettingsStateMode(d.pop("mode"))




        status = TranscriptionSettingsStateStatus(d.pop("status"))




        endpoint = d.pop("endpoint")

        model = d.pop("model")

        language = d.pop("language")

        version = d.pop("version")

        _workers = d.pop("workers", UNSET)
        workers: list[WorkerTranscriptionState] | Unset = UNSET
        if _workers is not UNSET:
            workers = []
            for workers_item_data in _workers:
                workers_item = WorkerTranscriptionState.from_dict(workers_item_data)



                workers.append(workers_item)


        transcription_settings_state = cls(
            mode=mode,
            status=status,
            endpoint=endpoint,
            model=model,
            language=language,
            version=version,
            workers=workers,
        )


        transcription_settings_state.additional_properties = d
        return transcription_settings_state

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
