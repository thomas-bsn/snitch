from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkerTranscriptionState")



@_attrs_define
class WorkerTranscriptionState:
    """
        Attributes:
            id (str):
            name (str):
            received_version (str):
            tested_version (str):
            active_version (str):
            error_code (str):
            test_requested (bool | Unset):
     """

    id: str
    name: str
    received_version: str
    tested_version: str
    active_version: str
    error_code: str
    test_requested: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        received_version = self.received_version

        tested_version = self.tested_version

        active_version = self.active_version

        error_code = self.error_code

        test_requested = self.test_requested


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "id": id,
            "name": name,
            "received_version": received_version,
            "tested_version": tested_version,
            "active_version": active_version,
            "error_code": error_code,
        })
        if test_requested is not UNSET:
            field_dict["test_requested"] = test_requested

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        received_version = d.pop("received_version")

        tested_version = d.pop("tested_version")

        active_version = d.pop("active_version")

        error_code = d.pop("error_code")

        test_requested = d.pop("test_requested", UNSET)

        worker_transcription_state = cls(
            id=id,
            name=name,
            received_version=received_version,
            tested_version=tested_version,
            active_version=active_version,
            error_code=error_code,
            test_requested=test_requested,
        )


        worker_transcription_state.additional_properties = d
        return worker_transcription_state

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
