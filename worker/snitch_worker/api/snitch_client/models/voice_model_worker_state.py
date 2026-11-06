from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.voice_model_worker_state_cause_code import VoiceModelWorkerStateCauseCode
from ..models.voice_model_worker_state_state import VoiceModelWorkerStateState






T = TypeVar("T", bound="VoiceModelWorkerState")



@_attrs_define
class VoiceModelWorkerState:
    """
        Attributes:
            worker_id (str):
            name (str):
            connected (bool):
            state (VoiceModelWorkerStateState):
            cause_code (VoiceModelWorkerStateCauseCode):
            bytes_received (int):
            bytes_total (int):
     """

    worker_id: str
    name: str
    connected: bool
    state: VoiceModelWorkerStateState
    cause_code: VoiceModelWorkerStateCauseCode
    bytes_received: int
    bytes_total: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        worker_id = self.worker_id

        name = self.name

        connected = self.connected

        state = self.state.value

        cause_code = self.cause_code.value

        bytes_received = self.bytes_received

        bytes_total = self.bytes_total


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "worker_id": worker_id,
            "name": name,
            "connected": connected,
            "state": state,
            "cause_code": cause_code,
            "bytes_received": bytes_received,
            "bytes_total": bytes_total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        worker_id = d.pop("worker_id")

        name = d.pop("name")

        connected = d.pop("connected")

        state = VoiceModelWorkerStateState(d.pop("state"))




        cause_code = VoiceModelWorkerStateCauseCode(d.pop("cause_code"))




        bytes_received = d.pop("bytes_received")

        bytes_total = d.pop("bytes_total")

        voice_model_worker_state = cls(
            worker_id=worker_id,
            name=name,
            connected=connected,
            state=state,
            cause_code=cause_code,
            bytes_received=bytes_received,
            bytes_total=bytes_total,
        )


        voice_model_worker_state.additional_properties = d
        return voice_model_worker_state

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
