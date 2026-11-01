from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_overview_worker_voice_state import VoiceOverviewWorkerVoiceState
from typing import cast

if TYPE_CHECKING:
  from ..models.worker_model import WorkerModel





T = TypeVar("T", bound="VoiceOverviewWorker")



@_attrs_define
class VoiceOverviewWorker:
    """
        Attributes:
            id (str):
            name (str):
            connected (bool):
            voice_state (VoiceOverviewWorkerVoiceState): ready : détient les modèles actifs et l'inférence de test a réussi
                ; untested : détient les modèles sans preuve d'inférence ; unavailable : un téléchargement ou un test a échoué
                (voir cause_code) ; models_missing : ne déclare pas les modèles actifs.
            cause_code (str):
            diarization (None | WorkerModel):
            fingerprint (None | WorkerModel):
            holds_active_diarization (bool):
            holds_active_fingerprint (bool):
     """

    id: str
    name: str
    connected: bool
    voice_state: VoiceOverviewWorkerVoiceState
    cause_code: str
    diarization: None | WorkerModel
    fingerprint: None | WorkerModel
    holds_active_diarization: bool
    holds_active_fingerprint: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        from ..models.worker_model import WorkerModel # noqa: PLC0415
        id = self.id

        name = self.name

        connected = self.connected

        voice_state = self.voice_state.value

        cause_code = self.cause_code

        diarization: dict[str, Any] | None
        if isinstance(self.diarization, WorkerModel):
            diarization = self.diarization.to_dict()
        else:
            diarization = self.diarization

        fingerprint: dict[str, Any] | None
        if isinstance(self.fingerprint, WorkerModel):
            fingerprint = self.fingerprint.to_dict()
        else:
            fingerprint = self.fingerprint

        holds_active_diarization = self.holds_active_diarization

        holds_active_fingerprint = self.holds_active_fingerprint


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "connected": connected,
            "voice_state": voice_state,
            "cause_code": cause_code,
            "diarization": diarization,
            "fingerprint": fingerprint,
            "holds_active_diarization": holds_active_diarization,
            "holds_active_fingerprint": holds_active_fingerprint,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worker_model import WorkerModel # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        connected = d.pop("connected")

        voice_state = VoiceOverviewWorkerVoiceState(d.pop("voice_state"))




        cause_code = d.pop("cause_code")

        def _parse_diarization(data: object) -> None | WorkerModel:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                diarization_type_1 = WorkerModel.from_dict(data)



                return diarization_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | WorkerModel, data)

        diarization = _parse_diarization(d.pop("diarization"))


        def _parse_fingerprint(data: object) -> None | WorkerModel:
            if data is None:
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                fingerprint_type_1 = WorkerModel.from_dict(data)



                return fingerprint_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | WorkerModel, data)

        fingerprint = _parse_fingerprint(d.pop("fingerprint"))


        holds_active_diarization = d.pop("holds_active_diarization")

        holds_active_fingerprint = d.pop("holds_active_fingerprint")

        voice_overview_worker = cls(
            id=id,
            name=name,
            connected=connected,
            voice_state=voice_state,
            cause_code=cause_code,
            diarization=diarization,
            fingerprint=fingerprint,
            holds_active_diarization=holds_active_diarization,
            holds_active_fingerprint=holds_active_fingerprint,
        )


        voice_overview_worker.additional_properties = d
        return voice_overview_worker

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
