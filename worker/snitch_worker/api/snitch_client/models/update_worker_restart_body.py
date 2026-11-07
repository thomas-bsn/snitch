from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.update_worker_restart_body_status import UpdateWorkerRestartBodyStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="UpdateWorkerRestartBody")



@_attrs_define
class UpdateWorkerRestartBody:
    """
        Attributes:
            status (UpdateWorkerRestartBodyStatus):
            result (str | Unset):
     """

    status: UpdateWorkerRestartBodyStatus
    result: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        result = self.result


        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({
            "status": status,
        })
        if result is not UNSET:
            field_dict["result"] = result

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = UpdateWorkerRestartBodyStatus(d.pop("status"))




        result = d.pop("result", UNSET)

        update_worker_restart_body = cls(
            status=status,
            result=result,
        )


        update_worker_restart_body.additional_properties = d
        return update_worker_restart_body

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
