from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VoiceOverviewActiveModels")



@_attrs_define
class VoiceOverviewActiveModels:
    """
        Attributes:
            diarization (str):
            fingerprint (str):
     """

    diarization: str
    fingerprint: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        diarization = self.diarization

        fingerprint = self.fingerprint


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "diarization": diarization,
            "fingerprint": fingerprint,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        diarization = d.pop("diarization")

        fingerprint = d.pop("fingerprint")

        voice_overview_active_models = cls(
            diarization=diarization,
            fingerprint=fingerprint,
        )


        voice_overview_active_models.additional_properties = d
        return voice_overview_active_models

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
