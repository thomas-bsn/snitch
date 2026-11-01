from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.worker_install_voice_state import WorkerInstallVoiceState






T = TypeVar("T", bound="WorkerInstallVoice")



@_attrs_define
class WorkerInstallVoice:
    """
        Attributes:
            state (WorkerInstallVoiceState): ready seulement après une inférence synthétique réussie.
            cause_code (str):
     """

    state: WorkerInstallVoiceState
    cause_code: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        cause_code = self.cause_code


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "state": state,
            "cause_code": cause_code,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = WorkerInstallVoiceState(d.pop("state"))




        cause_code = d.pop("cause_code")

        worker_install_voice = cls(
            state=state,
            cause_code=cause_code,
        )


        worker_install_voice.additional_properties = d
        return worker_install_voice

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
